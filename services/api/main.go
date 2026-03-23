// Package main implements the OnChain Health Monitor public REST API service.
// It consumes HealthEvent messages from the Kafka topic "onchain.health" to
// keep its in-memory protocol state current, and exposes that state via a
// JSON REST API.
//
// HTTP endpoints:
//
//   - GET /health                -> {"status":"ok"}
//   - GET /metrics               -> Real Prometheus metrics via promhttp
//   - GET /api/v1/protocols      -> list of monitored protocols with health scores
//   - GET /api/v1/protocols/{id} -> single protocol by ID
//
// Environment variables:
//
//   - KAFKA_BROKERS Comma-separated broker list (default: kafka:9092)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

// Protocol represents a DeFi protocol with health metadata.
type Protocol struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Chain       string    `json:"chain"`
	HealthScore int       `json:"health_score"`
	Status      string    `json:"status"`
	TVL         float64   `json:"tvl_usd"`
	Price       float64   `json:"price_usd"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// HealthEvent is the message consumed from the "onchain.health" topic.
type HealthEvent struct {
	ProtocolID string    `json:"protocol_id"`
	Score      int       `json:"score"`
	Label      string    `json:"label"`
	PriceUSD   float64   `json:"price_usd"`
	TVLUSD     float64   `json:"tvl_usd"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// statusFromScore returns a human-readable status label.
func statusFromScore(score int) string {
	switch {
	case score >= 70:
		return "healthy"
	case score >= 40:
		return "degraded"
	default:
		return "critical"
	}
}

var (
	mu        sync.RWMutex
	protocols = map[string]*Protocol{
		"uniswap": {
			ID:          "uniswap",
			Name:        "Uniswap",
			Category:    "DEX",
			Chain:       "Ethereum",
			HealthScore: 50,
			Status:      statusFromScore(50),
			TVL:         4_200_000_000,
			Price:       6.50,
			UpdatedAt:   time.Now().UTC(),
		},
		"aave": {
			ID:          "aave",
			Name:        "Aave",
			Category:    "Lending",
			Chain:       "Ethereum",
			HealthScore: 50,
			Status:      statusFromScore(50),
			TVL:         6_100_000_000,
			Price:       95.00,
			UpdatedAt:   time.Now().UTC(),
		},
		"compound": {
			ID:          "compound",
			Name:        "Compound",
			Category:    "Lending",
			Chain:       "Ethereum",
			HealthScore: 50,
			Status:      statusFromScore(50),
			TVL:         2_300_000_000,
			Price:       52.00,
			UpdatedAt:   time.Now().UTC(),
		},
	}
)

// Prometheus metrics
var (
	registry *prometheus.Registry

	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onchain_api_requests_total",
			Help: "Total HTTP requests handled by the API service.",
		},
		[]string{"method", "path", "status_code"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "onchain_api_request_duration_seconds",
			Help:    "Duration of HTTP requests handled by the API service.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "onchain_api_active_connections",
			Help: "Number of in-flight HTTP requests currently being handled.",
		},
	)

	inFlight int64
)

func init() {
	registry = prometheus.NewRegistry()
	registry.MustRegister(requestsTotal, requestDuration, activeConnections)
}

// getEnv returns the environment variable value or a fallback.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// consumeLoop reads HealthEvents from Kafka and updates the in-memory
// protocol state so HTTP responses always reflect the latest scores.
func consumeLoop(ctx context.Context, reader *kafka.Reader) {
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("kafka read error: %v", err)
			continue
		}

		var he HealthEvent
		if err := json.Unmarshal(msg.Value, &he); err != nil {
			log.Printf("unmarshal error: %v", err)
			continue
		}

		mu.Lock()
		if p, ok := protocols[he.ProtocolID]; ok {
			p.HealthScore = he.Score
			p.Status = statusFromScore(he.Score)
			p.Price = he.PriceUSD
			p.TVL = he.TVLUSD
			p.UpdatedAt = he.UpdatedAt
		}
		mu.Unlock()
	}
}

// instrumentedHandler wraps an http.HandlerFunc, recording request metrics.
func instrumentedHandler(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&inFlight, 1)
		activeConnections.Set(float64(atomic.LoadInt64(&inFlight)))
		defer func() {
			atomic.AddInt64(&inFlight, -1)
			activeConnections.Set(float64(atomic.LoadInt64(&inFlight)))
		}()

		start := time.Now()
		rw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(rw, r)

		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(rw.statusCode)

		requestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		requestDuration.WithLabelValues(r.Method, path).Observe(duration)
	}
}

// statusResponseWriter captures the HTTP status code written by a handler.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

func protocolsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/protocols")
	path = strings.Trim(path, "/")

	if path != "" {
		mu.RLock()
		defer mu.RUnlock()
		if p, ok := protocols[path]; ok {
			writeJSON(w, http.StatusOK, p)
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("protocol %q not found", path),
		})
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	list := make([]*Protocol, 0, len(protocols))
	for _, p := range protocols {
		list = append(list, p)
	}

	type listResponse struct {
		Protocols []*Protocol `json:"protocols"`
		Total     int         `json:"total"`
	}
	writeJSON(w, http.StatusOK, listResponse{
		Protocols: list,
		Total:     len(list),
	})
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[api] ")
	log.Println("Starting API service on :8080")

	ctx := context.Background()

	brokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ",")
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          "onchain.health",
		GroupID:        "api-group",
		MinBytes:       1,
		MaxBytes:       1 << 20, // 1 MiB
		CommitInterval: time.Second,
	})
	defer reader.Close()

	log.Printf("Kafka consumer connected to %v, topic=onchain.health group=api-group", brokers)

	go consumeLoop(ctx, reader)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/api/v1/protocols", instrumentedHandler("/api/v1/protocols", protocolsHandler))
	mux.HandleFunc("/api/v1/protocols/", instrumentedHandler("/api/v1/protocols/{id}", protocolsHandler))

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
