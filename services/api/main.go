// Package main implements the OnChain Health Monitor public REST API service.
// It exposes processed DeFi protocol health data via a JSON REST API.
//
// HTTP endpoints:
//   - GET /health                    → {"status":"ok"}
//   - GET /metrics                   → Real Prometheus metrics via promhttp
//   - GET /api/v1/protocols          → list of monitored protocols with health scores
//   - GET /api/v1/protocols/{id}     → single protocol by ID
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	protocols = []*Protocol{
		{
			ID:          "uniswap",
			Name:        "Uniswap",
			Category:    "DEX",
			Chain:       "Ethereum",
			HealthScore: 82,
			Status:      "healthy",
			TVL:         4_200_000_000,
			Price:       6.52,
			UpdatedAt:   time.Now().UTC(),
		},
		{
			ID:          "aave",
			Name:        "Aave",
			Category:    "Lending",
			Chain:       "Ethereum",
			HealthScore: 76,
			Status:      "healthy",
			TVL:         6_100_000_000,
			Price:       96.10,
			UpdatedAt:   time.Now().UTC(),
		},
		{
			ID:          "compound",
			Name:        "Compound",
			Category:    "Lending",
			Chain:       "Ethereum",
			HealthScore: 41,
			Status:      "degraded",
			TVL:         2_300_000_000,
			Price:       51.80,
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

	// inFlight tracks the raw count atomically for the gauge.
	inFlight int64
)

func init() {
	registry = prometheus.NewRegistry()
	registry.MustRegister(requestsTotal, requestDuration, activeConnections)
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
	// Strip prefix and route to single-protocol handler.
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/protocols")
	path = strings.Trim(path, "/")

	if path != "" {
		// /api/v1/protocols/{id}
		mu.RLock()
		defer mu.RUnlock()
		for _, p := range protocols {
			if p.ID == path {
				writeJSON(w, http.StatusOK, p)
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("protocol %q not found", path),
		})
		return
	}

	// /api/v1/protocols
	mu.RLock()
	defer mu.RUnlock()
	type listResponse struct {
		Protocols []*Protocol `json:"protocols"`
		Total     int         `json:"total"`
	}
	writeJSON(w, http.StatusOK, listResponse{
		Protocols: protocols,
		Total:     len(protocols),
	})
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[api] ")
	log.Println("Starting API service on :8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// Instrumented routes.
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
