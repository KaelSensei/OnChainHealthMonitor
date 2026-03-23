// Package main implements the OnChain Health Monitor notifier service.
// It consumes HealthEvent messages from the Kafka topic "onchain.health"
// and fires alerts whenever a protocol score drops below the critical
// threshold (30).
//
// HTTP endpoints:
//
//   - GET /health  -> {"status":"ok"}
//   - GET /metrics -> Real Prometheus metrics via promhttp
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
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

const criticalThreshold = 30

// HealthEvent is the message consumed from the "onchain.health" topic.
type HealthEvent struct {
	ProtocolID string    `json:"protocol_id"`
	Score      int       `json:"score"`
	Label      string    `json:"label"`
	PriceUSD   float64   `json:"price_usd"`
	TVLUSD     float64   `json:"tvl_usd"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Alert represents a triggered alert.
type Alert struct {
	Protocol string
	Score    int
	Severity string
	Message  string
	FiredAt  time.Time
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

// getEnv returns the environment variable value or a fallback.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func severity(score int) string {
	if score < 20 {
		return "CRITICAL"
	}
	return "WARNING"
}

func sendNotification(a *Alert) {
	start := time.Now()

	log.Printf("ALERT %s protocol=%s score=%d message=%q",
		a.Severity, a.Protocol, a.Score, a.Message)
	notificationsSentTotal.WithLabelValues(a.Protocol, "log").Inc()

	if a.Score < 20 {
		log.Printf("WEBHOOK fired for protocol=%s score=%d", a.Protocol, a.Score)
		notificationsSentTotal.WithLabelValues(a.Protocol, "webhook").Inc()
	}

	lastAlertScore.WithLabelValues(a.Protocol).Set(float64(a.Score))
	notificationDuration.Observe(time.Since(start).Seconds())
}

// consumeLoop reads HealthEvents from Kafka and fires alerts when the score
// drops below criticalThreshold.
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

		if he.Score < criticalThreshold {
			sev := severity(he.Score)
			sendNotification(&Alert{
				Protocol: he.ProtocolID,
				Score:    he.Score,
				Severity: sev,
				Message: fmt.Sprintf(
					"[%s] Protocol %q health score %d/100 is below threshold %d",
					sev, he.ProtocolID, he.Score, criticalThreshold,
				),
				FiredAt: time.Now().UTC(),
			})
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

	ctx := context.Background()

	brokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ",")
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          "onchain.health",
		GroupID:        "notifier-group",
		MinBytes:       1,
		MaxBytes:       1 << 20, // 1 MiB
		CommitInterval: time.Second,
	})
	defer reader.Close()

	log.Printf("Kafka consumer connected to %v, topic=onchain.health group=notifier-group", brokers)

	go consumeLoop(ctx, reader)

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
