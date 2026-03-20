// Package main implements the OnChain Health Monitor notifier service.
// It simulates alerting by polling internal health scores every 5 seconds,
// logging a formatted alert whenever a score drops below the critical threshold (30).
//
// HTTP endpoints:
//   - GET /health  → {"status":"ok"}
//   - GET /metrics → Real Prometheus metrics via promhttp
package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const criticalThreshold = 30

// Alert represents a triggered alert.
type Alert struct {
	Protocol string
	Score    int
	Severity string
	Message  string
	FiredAt  time.Time
	Resolved bool
}


// Prometheus metrics
var (
	registry *prometheus.Registry

	notificationsSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onchain_notifier_notifications_sent_total",
			Help: "Total notifications sent by the notifier.",
		},
		[]string{"protocol", "channel"},
	)

	lastAlertScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "onchain_notifier_last_alert_score",
			Help: "Health score that triggered the most recent alert for each protocol.",
		},
		[]string{"protocol"},
	)

	notificationDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "onchain_notifier_notification_duration_seconds",
			Help:    "Time taken to process and send one notification.",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	registry = prometheus.NewRegistry()
	registry.MustRegister(notificationsSentTotal, lastAlertScore, notificationDuration)
}

// simulatedScores mirrors what the analyzer would publish.
// In a real system this would be consumed from a message bus or HTTP endpoint.
func randomScore(protocol string) int {
	// Bias compound lower to trigger alerts more visibly.
	base := map[string]int{
		"uniswap":  65,
		"aave":     70,
		"compound": 35,
	}
	b := base[protocol]
	score := b + rand.Intn(31) - 15
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func severity(score int) string {
	if score < 20 {
		return "CRITICAL"
	}
	return "WARNING"
}

func sendNotification(a *Alert) {
	start := time.Now()

	// Log channel (always).
	log.Printf("🔔 ALERT %s protocol=%s score=%d message=%q",
		a.Severity, a.Protocol, a.Score, a.Message)
	notificationsSentTotal.WithLabelValues(a.Protocol, "log").Inc()

	// Simulate webhook channel for critical alerts.
	if a.Score < 20 {
		log.Printf("📡 WEBHOOK fired for protocol=%s score=%d", a.Protocol, a.Score)
		notificationsSentTotal.WithLabelValues(a.Protocol, "webhook").Inc()
	}

	lastAlertScore.WithLabelValues(a.Protocol).Set(float64(a.Score))
	notificationDuration.Observe(time.Since(start).Seconds())
}

func alertLoop() {
	protocols := []string{"uniswap", "aave", "compound"}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var fired []*Alert
		for _, p := range protocols {
			score := randomScore(p)
			if score < criticalThreshold {
				a := &Alert{
					Protocol: p,
					Score:    score,
					Severity: severity(score),
					Message: fmt.Sprintf(
						"[%s] Protocol %q health score %d/100 is below threshold %d",
						severity(score), p, score, criticalThreshold,
					),
					FiredAt: time.Now().UTC(),
				}
				fired = append(fired, a)
				sendNotification(a)
			}
		}

		if len(fired) == 0 {
			log.Printf("✅ All protocols healthy (no alerts)")
		}
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[notifier] ")
	log.Printf("Starting notifier service on :8083 (alert threshold: score < %d)", criticalThreshold)

	go alertLoop()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:         ":8083",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
