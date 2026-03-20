// Package main implements the OnChain Health Monitor collector service.
// It runs in mock mode by default, emitting fake but realistic DeFi events
// (prices, TVL, protocol events) as JSON every 2 seconds.
//
// HTTP endpoints:
//   - GET /health  → {"status":"ok"}
//   - GET /metrics → Real Prometheus metrics via promhttp
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Protocol represents a monitored DeFi protocol.
type Protocol struct {
	ID   string
	Name string
}

// DeFiEvent represents a single collected data point from a DeFi protocol.
type DeFiEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	ProtocolID   string    `json:"protocol_id"`
	ProtocolName string    `json:"protocol_name"`
	Price        float64   `json:"price_usd"`
	TVL          float64   `json:"tvl_usd"`
	EventType    string    `json:"event_type"`
	Volume24h    float64   `json:"volume_24h_usd"`
}

var protocols = []Protocol{
	{ID: "uniswap", Name: "Uniswap"},
	{ID: "aave", Name: "Aave"},
	{ID: "compound", Name: "Compound"},
}

// State holds per-protocol running state to simulate realistic drift.
type protocolState struct {
	price float64
	tvl   float64
}

var (
	mu     sync.RWMutex
	states map[string]*protocolState

	// Baseline values per protocol (USD).
	baselines = map[string][2]float64{
		"uniswap":  {6.50, 4_200_000_000},
		"aave":     {95.00, 6_100_000_000},
		"compound": {52.00, 2_300_000_000},
	}
)

// Prometheus metrics
var (
	registry *prometheus.Registry

	eventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onchain_collector_events_total",
			Help: "Total DeFi events emitted by the collector.",
		},
		[]string{"protocol", "event_type"},
	)

	eventGenerationDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "onchain_collector_event_generation_duration_seconds",
			Help:    "Time taken to generate one mock DeFi event.",
			Buckets: prometheus.DefBuckets,
		},
	)

	lastPrice = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "onchain_collector_last_price",
			Help: "Current mock price in USD for each protocol.",
		},
		[]string{"protocol"},
	)

	lastTVL = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "onchain_collector_last_tvl_usd",
			Help: "Current mock TVL in USD for each protocol.",
		},
		[]string{"protocol"},
	)
)

func init() {
	states = make(map[string]*protocolState)
	for _, p := range protocols {
		b := baselines[p.ID]
		states[p.ID] = &protocolState{price: b[0], tvl: b[1]}
	}

	registry = prometheus.NewRegistry()
	registry.MustRegister(eventsTotal, eventGenerationDuration, lastPrice, lastTVL)
}

// getEnv returns the environment variable value or a fallback.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// initTracer sets up the OpenTelemetry tracer provider with OTLP gRPC exporter.
func initTracer(ctx context.Context, serviceName string) (func(), error) {
	conn, err := grpc.NewClient(
		getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "jaeger:4317"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func() {
		_ = tp.Shutdown(context.Background())
	}, nil
}

// drift applies a small random walk to a value, clamped to ±20% of baseline.
func drift(current, baseline, maxPctChange float64) float64 {
	pct := (rand.Float64()*2 - 1) * maxPctChange
	next := current * (1 + pct)
	lo := baseline * 0.80
	hi := baseline * 1.20
	return math.Max(lo, math.Min(hi, next))
}

// normaliseEventType maps internal event types to Prometheus label-safe values.
func normaliseEventType(raw string) string {
	switch raw {
	case "price_update", "tvl_change":
		return raw
	default:
		return "protocol_event"
	}
}

func pickEventType() string {
	types := []string{"price_update", "tvl_change", "swap", "liquidation", "deposit"}
	return types[rand.Intn(len(types))]
}

func generateEvent(ctx context.Context, p Protocol) DeFiEvent {
	tracer := otel.Tracer("collector")

	start := time.Now()

	mu.Lock()
	s := states[p.ID]
	b := baselines[p.ID]
	s.price = drift(s.price, b[0], 0.02)
	s.tvl = drift(s.tvl, b[1], 0.01)
	price := s.price
	tvl := s.tvl
	mu.Unlock()

	volume := tvl * (0.01 + rand.Float64()*0.04)
	eventType := pickEventType()

	// Create a span for this event generation.
	_, span := tracer.Start(ctx, "generate_event", trace.WithAttributes(
		attribute.String("protocol", p.ID),
		attribute.String("event_type", eventType),
		attribute.Float64("price_usd", math.Round(price*10000)/10000),
		attribute.Float64("tvl_usd", math.Round(tvl)),
	))
	defer span.End()

	ev := DeFiEvent{
		Timestamp:    time.Now().UTC(),
		ProtocolID:   p.ID,
		ProtocolName: p.Name,
		Price:        math.Round(price*10000) / 10000,
		TVL:          math.Round(tvl),
		EventType:    eventType,
		Volume24h:    math.Round(volume),
	}

	// Update Prometheus metrics
	eventsTotal.WithLabelValues(p.ID, normaliseEventType(eventType)).Inc()
	lastPrice.WithLabelValues(p.ID).Set(price)
	lastTVL.WithLabelValues(p.ID).Set(tvl)
	eventGenerationDuration.Observe(time.Since(start).Seconds())

	return ev
}

func emitLoop() {
	ctx := context.Background()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	enc := json.NewEncoder(log.Writer())
	enc.SetIndent("", "  ")
	for range ticker.C {
		for _, p := range protocols {
			ev := generateEvent(ctx, p)
			if err := enc.Encode(ev); err != nil {
				log.Printf("encode error: %v", err)
			}
		}
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[collector] ")
	log.Println("Starting collector service on :8081 (mock mode)")

	ctx := context.Background()
	shutdown, err := initTracer(ctx, "onchain-collector")
	if err != nil {
		log.Printf("WARNING: failed to initialize tracer: %v - continuing without tracing", err)
	} else {
		defer shutdown()
		log.Println("OpenTelemetry tracer initialized")
	}

	go emitLoop()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
