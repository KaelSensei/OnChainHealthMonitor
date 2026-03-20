// Package main implements the OnChain Health Monitor analyzer service.
// It simulates receiving events from the collector and producing health scores
// (0-100) per protocol, logging results every 3 seconds.
//
// HTTP endpoints:
//   - GET /health  → {"status":"ok"}
//   - GET /metrics → Real Prometheus metrics via promhttp
package main

import (
	"context"
	"fmt"
	"log"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// HealthScore holds the computed health score for a protocol.
type HealthScore struct {
	ProtocolID string
	Score      int
	Label      string
	UpdatedAt  time.Time
}

var (
	mu     sync.RWMutex
	scores = map[string]*HealthScore{
		"uniswap":  {ProtocolID: "uniswap", Score: 80, Label: "healthy"},
		"aave":     {ProtocolID: "aave", Score: 75, Label: "healthy"},
		"compound": {ProtocolID: "compound", Score: 60, Label: "degraded"},
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
			Help:    "Time taken for one full analysis cycle across all protocols.",
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

	// Seed gauges with initial values.
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

// analyzeLoop simulates receiving events and computing health scores.
func analyzeLoop() {
	tracer := otel.Tracer("analyzer")
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		start := time.Now()

		mu.Lock()
		for id, hs := range scores {
			ctx, span := tracer.Start(context.Background(), "analyze_protocol",
				// attributes added after computation below
			)

			// Simulate score drift: small random walk clamped 0–100.
			delta := rand.Intn(11) - 5 // -5 to +5
			next := hs.Score + delta
			if next < 0 {
				next = 0
			} else if next > 100 {
				next = 100
			}
			hs.Score = next
			hs.Label = scoreLabel(next)
			hs.UpdatedAt = time.Now().UTC()

			log.Printf("protocol=%s score=%d label=%s", id, next, hs.Label)

			// Add span attributes.
			span.SetAttributes(
				attribute.String("protocol", id),
				attribute.Int("health_score", next),
				attribute.String("severity", hs.Label),
			)
			span.End()
			_ = ctx // ctx available for downstream propagation if needed

			// Update Prometheus metrics.
			scoresComputedTotal.WithLabelValues(id).Inc()
			healthScore.WithLabelValues(id).Set(float64(next))

			if next < 40 {
				sev := alertSeverity(next)
				alertsTriggeredTotal.WithLabelValues(id, sev).Inc()
			}
		}
		mu.Unlock()

		analysisDuration.Observe(time.Since(start).Seconds())
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

	go analyzeLoop()

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
