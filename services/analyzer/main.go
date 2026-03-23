// Package main implements the OnChain Health Monitor analyzer service.
// It consumes DeFiEvent messages from the Kafka topic "onchain.events",
// computes a health score (0-100) per protocol based on price and TVL
// deviation from their baselines, and publishes HealthEvent messages to
// the Kafka topic "onchain.health".
//
// HTTP endpoints:
//
//   - GET /health  -> {"status":"ok"}
//   - GET /metrics -> Real Prometheus metrics via promhttp
//
// Environment variables:
//
//   - KAFKA_BROKERS              Comma-separated broker list (default: kafka:9092)
//   - OTEL_EXPORTER_OTLP_ENDPOINT gRPC endpoint for traces  (default: jaeger:4317)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DeFiEvent is the message consumed from the "onchain.events" topic.
type DeFiEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	ProtocolID   string    `json:"protocol_id"`
	ProtocolName string    `json:"protocol_name"`
	Price        float64   `json:"price_usd"`
	TVL          float64   `json:"tvl_usd"`
	EventType    string    `json:"event_type"`
	Volume24h    float64   `json:"volume_24h_usd"`
}

// HealthEvent is the message published to the "onchain.health" topic.
type HealthEvent struct {
	ProtocolID string    `json:"protocol_id"`
	Score      int       `json:"score"`
	Label      string    `json:"label"`
	PriceUSD   float64   `json:"price_usd"`
	TVLUSD     float64   `json:"tvl_usd"`
	UpdatedAt  time.Time `json:"updated_at"`
}

var (
	mu     sync.RWMutex
	scores = map[string]*HealthEvent{
		"uniswap":  {ProtocolID: "uniswap", Score: 50, Label: "degraded"},
		"aave":     {ProtocolID: "aave", Score: 50, Label: "degraded"},
		"compound": {ProtocolID: "compound", Score: 50, Label: "degraded"},
	}

	// baselines holds the expected price (USD) and TVL (USD) per protocol.
	// Score = 50 at baseline, 100 at +20%, 0 at -20%.
	baselines = map[string][2]float64{
		"uniswap":  {6.50, 4_200_000_000},
		"aave":     {95.00, 6_100_000_000},
		"compound": {52.00, 2_300_000_000},
	}
)

// Prometheus metrics
var (
	registry *prometheus.Registry

	scoresComputedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onchain_analyzer_scores_computed_total",
			Help: "Total number of health score computations per protocol.",
		},
		[]string{"protocol"},
	)

	healthScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "onchain_analyzer_health_score",
			Help: "Current health score (0-100) for each protocol.",
		},
		[]string{"protocol"},
	)

	alertsTriggeredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onchain_analyzer_alerts_triggered_total",
			Help: "Total number of alerts triggered by the analyzer.",
		},
		[]string{"protocol", "severity"},
	)

	analysisDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "onchain_analyzer_analysis_duration_seconds",
			Help:    "Time taken to process one event and compute a health score.",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	registry = prometheus.NewRegistry()
	registry.MustRegister(
		scoresComputedTotal,
		healthScore,
		alertsTriggeredTotal,
		analysisDuration,
	)

	for id, hs := range scores {
		healthScore.WithLabelValues(id).Set(float64(hs.Score))
	}
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

func scoreLabel(score int) string {
	switch {
	case score >= 70:
		return "healthy"
	case score >= 40:
		return "degraded"
	default:
		return "critical"
	}
}

func alertSeverity(score int) string {
	if score < 20 {
		return "critical"
	}
	return "warning"
}

// computeScore maps price and TVL deviation from baseline to a 0-100 score.
// At baseline the score is 50. At +20% it is 100, at -20% it is 0.
// Liquidation events apply an additional -10 penalty.
func computeScore(protocolID string, price, tvl float64, eventType string) int {
	b := baselines[protocolID]
	priceBaseline, tvlBaseline := b[0], b[1]

	priceNorm := (price - priceBaseline*0.8) / (priceBaseline * 0.4)
	tvlNorm := (tvl - tvlBaseline*0.8) / (tvlBaseline * 0.4)

	priceNorm = math.Max(0, math.Min(1, priceNorm))
	tvlNorm = math.Max(0, math.Min(1, tvlNorm))

	score := int((priceNorm*0.5+tvlNorm*0.5)*100)

	if eventType == "liquidation" {
		score -= 10
	}

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// consumeLoop reads DeFiEvents from Kafka, computes health scores, and
// publishes HealthEvents back to Kafka.
func consumeLoop(ctx context.Context, reader *kafka.Reader, writer *kafka.Writer) {
	tracer := otel.Tracer("analyzer")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("kafka read error: %v", err)
			continue
		}

		start := time.Now()

		var ev DeFiEvent
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			log.Printf("unmarshal error: %v", err)
			continue
		}

		_, span := tracer.Start(ctx, "analyze_protocol")
		score := computeScore(ev.ProtocolID, ev.Price, ev.TVL, ev.EventType)
		label := scoreLabel(score)

		he := &HealthEvent{
			ProtocolID: ev.ProtocolID,
			Score:      score,
			Label:      label,
			PriceUSD:   ev.Price,
			TVLUSD:     ev.TVL,
			UpdatedAt:  time.Now().UTC(),
		}

		span.SetAttributes(
			attribute.String("protocol", ev.ProtocolID),
			attribute.Int("health_score", score),
			attribute.String("label", label),
		)
		span.End()

		mu.Lock()
		scores[ev.ProtocolID] = he
		mu.Unlock()

		log.Printf("protocol=%s score=%d label=%s price=%.4f tvl=%.0f",
			ev.ProtocolID, score, label, ev.Price, ev.TVL)

		scoresComputedTotal.WithLabelValues(ev.ProtocolID).Inc()
		healthScore.WithLabelValues(ev.ProtocolID).Set(float64(score))
		if score < 40 {
			alertsTriggeredTotal.WithLabelValues(ev.ProtocolID, alertSeverity(score)).Inc()
		}

		analysisDuration.Observe(time.Since(start).Seconds())

		payload, err := json.Marshal(he)
		if err != nil {
			log.Printf("marshal error: %v", err)
			continue
		}

		if err := writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(ev.ProtocolID),
			Value: payload,
		}); err != nil {
			log.Printf("kafka write error: %v", err)
		}
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[analyzer] ")
	log.Println("Starting analyzer service on :8082")

	ctx := context.Background()
	shutdown, err := initTracer(ctx, "onchain-analyzer")
	if err != nil {
		log.Printf("WARNING: failed to initialize tracer: %v - continuing without tracing", err)
	} else {
		defer shutdown()
		log.Println("OpenTelemetry tracer initialized")
	}

	brokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ",")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          "onchain.events",
		GroupID:        "analyzer-group",
		MinBytes:       1,
		MaxBytes:       1 << 20, // 1 MiB
		CommitInterval: time.Second,
	})
	defer reader.Close()

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        "onchain.health",
		Balancer:     &kafka.Hash{},
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}
	defer writer.Close()

	log.Printf("Kafka consumer connected to %v, topic=onchain.events group=analyzer-group", brokers)
	log.Printf("Kafka writer connected to %v, topic=onchain.health", brokers)

	go consumeLoop(ctx, reader, writer)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:         ":8082",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
