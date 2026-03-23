// Package main implements the OnChain Health Monitor notifier service.
// It consumes HealthEvent messages from the Kafka topic "onchain.health",
// fires system-level alerts when a score drops below the critical threshold (30),
// and routes per-user alerts to the RabbitMQ topic exchange "onchain.alerts"
// based on subscriptions stored in Redis.
//
// HTTP endpoints:
//
//   - GET /health  -> {"status":"ok"}
//   - GET /metrics -> Real Prometheus metrics via promhttp
//
// Environment variables:
//
//   - KAFKA_BROKERS  Comma-separated Kafka broker list   (default: kafka:9092)
//   - REDIS_ADDR     Redis address                       (default: redis:6379)
//   - RABBITMQ_URL   AMQP connection URL                 (default: amqp://onchain:onchain@rabbitmq:5672/)
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
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	kafka "github.com/segmentio/kafka-go"
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

// Subscription mirrors the data model stored in Redis by the subscription service.
type Subscription struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	ProtocolID string    `json:"protocol_id"`
	Threshold  int       `json:"threshold"`
	CreatedAt  time.Time `json:"created_at"`
}

// AlertMessage is the payload published to the RabbitMQ exchange.
type AlertMessage struct {
	UserID     string    `json:"user_id"`
	ProtocolID string    `json:"protocol_id"`
	Score      int       `json:"score"`
	Label      string    `json:"label"`
	Threshold  int       `json:"threshold"`
	Message    string    `json:"message"`
	FiredAt    time.Time `json:"fired_at"`
}

// Alert represents a system-level triggered alert.
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

	subscriptionAlertsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onchain_notifier_subscription_alerts_total",
			Help: "Total per-user subscription alerts routed to RabbitMQ.",
		},
		[]string{"protocol"},
	)
)

func init() {
	registry = prometheus.NewRegistry()
	registry.MustRegister(
		notificationsSentTotal,
		lastAlertScore,
		notificationDuration,
		subscriptionAlertsTotal,
	)
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

func sendSystemAlert(a *Alert) {
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

// routeSubscriptionAlerts looks up subscriptions in Redis whose threshold is
// >= the event score and publishes an AlertMessage to RabbitMQ for each match.
func routeSubscriptionAlerts(ctx context.Context, rdb *redis.Client, ch *amqp.Channel, he *HealthEvent) {
	ids, err := rdb.SMembers(ctx, "proto_subs:"+he.ProtocolID).Result()
	if err != nil || len(ids) == 0 {
		return
	}

	for _, id := range ids {
		val, err := rdb.Get(ctx, "sub:"+id).Result()
		if err != nil {
			continue
		}
		var sub Subscription
		if err := json.Unmarshal([]byte(val), &sub); err != nil {
			continue
		}

		if he.Score > sub.Threshold {
			continue
		}

		msg := AlertMessage{
			UserID:     sub.UserID,
			ProtocolID: he.ProtocolID,
			Score:      he.Score,
			Label:      he.Label,
			Threshold:  sub.Threshold,
			Message: fmt.Sprintf(
				"Protocol %q health score %d/100 crossed your threshold of %d",
				he.ProtocolID, he.Score, sub.Threshold,
			),
			FiredAt: time.Now().UTC(),
		}

		payload, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		if err := ch.PublishWithContext(ctx,
			"onchain.alerts",
			"user."+sub.UserID,
			false, false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        payload,
			},
		); err != nil {
			log.Printf("rabbitmq publish error user=%s: %v", sub.UserID, err)
			continue
		}

		subscriptionAlertsTotal.WithLabelValues(he.ProtocolID).Inc()
		log.Printf("subscription alert routed user=%s protocol=%s score=%d threshold=%d",
			sub.UserID, he.ProtocolID, he.Score, sub.Threshold)
	}
}

// consumeLoop reads HealthEvents from Kafka, fires system alerts, and routes
// per-user subscription alerts via RabbitMQ.
func consumeLoop(ctx context.Context, reader *kafka.Reader, rdb *redis.Client, amqpCh *amqp.Channel) {
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

		// System-level alert (below hard threshold).
		if he.Score < criticalThreshold {
			sev := severity(he.Score)
			sendSystemAlert(&Alert{
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

		// Per-user subscription routing.
		routeSubscriptionAlerts(ctx, rdb, amqpCh, &he)
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

	// Kafka reader.
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ",")
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          "onchain.health",
		GroupID:        "notifier-group",
		MinBytes:       1,
		MaxBytes:       1 << 20,
		CommitInterval: time.Second,
	})
	defer reader.Close()
	log.Printf("Kafka consumer connected to %v, topic=onchain.health group=notifier-group", brokers)

	// Redis client.
	rdb := redis.NewClient(&redis.Options{
		Addr: getEnv("REDIS_ADDR", "redis:6379"),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	log.Printf("Redis connected at %s", getEnv("REDIS_ADDR", "redis:6379"))

	// RabbitMQ connection and channel.
	amqpConn, err := amqp.Dial(getEnv("RABBITMQ_URL", "amqp://onchain:onchain@rabbitmq:5672/"))
	if err != nil {
		log.Fatalf("rabbitmq connection failed: %v", err)
	}
	defer amqpConn.Close()

	amqpCh, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("amqp channel error: %v", err)
	}
	defer amqpCh.Close()

	if err := amqpCh.ExchangeDeclare(
		"onchain.alerts", "topic", true, false, false, false, nil,
	); err != nil {
		log.Fatalf("exchange declare error: %v", err)
	}
	log.Printf("RabbitMQ connected, exchange=onchain.alerts declared")

	go consumeLoop(ctx, reader, rdb, amqpCh)

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
