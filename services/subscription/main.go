// Package main implements the OnChain Health Monitor subscription service.
// Users create subscriptions pairing a protocol with a score threshold.
// When the notifier publishes a matching alert to RabbitMQ, this service
// pushes it to any WebSocket connections open for that user.
//
// HTTP endpoints:
//
//   - GET    /health                                  -> {"status":"ok"}
//   - GET    /metrics                                 -> Prometheus metrics
//   - POST   /api/v1/subscriptions                    -> create subscription
//   - GET    /api/v1/subscriptions/{user_id}          -> list subscriptions for user
//   - DELETE /api/v1/subscriptions/{user_id}/{id}     -> delete subscription
//   - GET    /ws?user_id={user_id}                    -> WebSocket alert stream
//
// Environment variables:
//
//   - REDIS_ADDR    Redis address                 (default: redis:6379)
//   - RABBITMQ_URL  AMQP connection URL           (default: amqp://onchain:onchain@rabbitmq:5672/)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

// Subscription represents a user's alert rule for one protocol.
type Subscription struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	ProtocolID string    `json:"protocol_id"`
	Threshold  int       `json:"threshold"`
	CreatedAt  time.Time `json:"created_at"`
}

// AlertMessage is pushed to WebSocket clients when an alert fires.
type AlertMessage struct {
	UserID     string    `json:"user_id"`
	ProtocolID string    `json:"protocol_id"`
	Score      int       `json:"score"`
	Label      string    `json:"label"`
	Threshold  int       `json:"threshold"`
	Message    string    `json:"message"`
	FiredAt    time.Time `json:"fired_at"`
}

var (
	rdb      *redis.Client
	amqpConn *amqp.Connection

	upgrader = websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}
)

// Prometheus metrics
var (
	registry *prometheus.Registry

	subsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "onchain_subscription_active_total",
			Help: "Total active subscriptions currently stored.",
		},
	)

	wsConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "onchain_subscription_ws_connections_active",
			Help: "Number of WebSocket connections currently open.",
		},
	)

	alertsDeliveredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onchain_subscription_alerts_delivered_total",
			Help: "Total alert messages delivered via WebSocket.",
		},
		[]string{"protocol"},
	)

	wsConnectionCount int64
)

func init() {
	registry = prometheus.NewRegistry()
	registry.MustRegister(subsTotal, wsConnectionsActive, alertsDeliveredTotal)
}

// getEnv returns the environment variable value or a fallback.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Redis key helpers.
func keySubscription(id string) string       { return "sub:" + id }
func keyUserSubs(userID string) string        { return "user_subs:" + userID }
func keyProtoSubs(protocolID string) string   { return "proto_subs:" + protocolID }

// saveSubscription writes a subscription to Redis.
func saveSubscription(ctx context.Context, sub *Subscription) error {
	payload, err := json.Marshal(sub)
	if err != nil {
		return err
	}
	pipe := rdb.Pipeline()
	pipe.Set(ctx, keySubscription(sub.ID), payload, 0)
	pipe.SAdd(ctx, keyUserSubs(sub.UserID), sub.ID)
	pipe.SAdd(ctx, keyProtoSubs(sub.ProtocolID), sub.ID)
	_, err = pipe.Exec(ctx)
	return err
}

// getSubscription fetches a single subscription by ID.
func getSubscription(ctx context.Context, id string) (*Subscription, error) {
	val, err := rdb.Get(ctx, keySubscription(id)).Result()
	if err != nil {
		return nil, err
	}
	var sub Subscription
	return &sub, json.Unmarshal([]byte(val), &sub)
}

// listUserSubscriptions returns all subscriptions for a user.
func listUserSubscriptions(ctx context.Context, userID string) ([]*Subscription, error) {
	ids, err := rdb.SMembers(ctx, keyUserSubs(userID)).Result()
	if err != nil {
		return nil, err
	}
	subs := make([]*Subscription, 0, len(ids))
	for _, id := range ids {
		sub, err := getSubscription(ctx, id)
		if err != nil {
			log.Printf("WARNING: dangling subscription id=%s: %v", id, err)
			continue
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// deleteSubscription removes a subscription from Redis.
func deleteSubscription(ctx context.Context, sub *Subscription) error {
	pipe := rdb.Pipeline()
	pipe.Del(ctx, keySubscription(sub.ID))
	pipe.SRem(ctx, keyUserSubs(sub.UserID), sub.ID)
	pipe.SRem(ctx, keyProtoSubs(sub.ProtocolID), sub.ID)
	_, err := pipe.Exec(ctx)
	return err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

// createHandler handles POST /api/v1/subscriptions.
func createHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID     string `json:"user_id"`
		ProtocolID string `json:"protocol_id"`
		Threshold  int    `json:"threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.UserID == "" || body.ProtocolID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id and protocol_id are required"})
		return
	}
	if body.Threshold <= 0 || body.Threshold > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "threshold must be between 1 and 100"})
		return
	}

	sub := &Subscription{
		ID:         uuid.New().String(),
		UserID:     body.UserID,
		ProtocolID: body.ProtocolID,
		Threshold:  body.Threshold,
		CreatedAt:  time.Now().UTC(),
	}

	if err := saveSubscription(r.Context(), sub); err != nil {
		log.Printf("redis error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save subscription"})
		return
	}

	subsTotal.Inc()
	writeJSON(w, http.StatusCreated, sub)
}

// listHandler handles GET /api/v1/subscriptions/{user_id}.
func listHandler(w http.ResponseWriter, r *http.Request, userID string) {
	subs, err := listUserSubscriptions(r.Context(), userID)
	if err != nil {
		log.Printf("redis error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list subscriptions"})
		return
	}
	type listResponse struct {
		Subscriptions []*Subscription `json:"subscriptions"`
		Total         int             `json:"total"`
	}
	writeJSON(w, http.StatusOK, listResponse{Subscriptions: subs, Total: len(subs)})
}

// deleteHandler handles DELETE /api/v1/subscriptions/{user_id}/{id}.
func deleteHandler(w http.ResponseWriter, r *http.Request, userID, id string) {
	sub, err := getSubscription(r.Context(), id)
	if err == redis.Nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "subscription not found"})
		return
	}
	if err != nil {
		log.Printf("redis error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch subscription"})
		return
	}
	if sub.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "subscription does not belong to this user"})
		return
	}
	if err := deleteSubscription(r.Context(), sub); err != nil {
		log.Printf("redis error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete subscription"})
		return
	}
	subsTotal.Dec()
	w.WriteHeader(http.StatusNoContent)
}

// subscriptionsHandler routes all /api/v1/subscriptions/* requests.
func subscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/subscriptions")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	switch {
	case path == "" && r.Method == http.MethodPost:
		createHandler(w, r)
	case len(parts) == 1 && parts[0] != "" && r.Method == http.MethodGet:
		listHandler(w, r, parts[0])
	case len(parts) == 2 && r.Method == http.MethodDelete:
		deleteHandler(w, r, parts[0], parts[1])
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// wsHandler handles GET /ws?user_id={user_id}.
// It declares a per-user queue on RabbitMQ, binds it to the onchain.alerts
// topic exchange, and streams incoming alert messages to the WebSocket client.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query param required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	atomic.AddInt64(&wsConnectionCount, 1)
	wsConnectionsActive.Set(float64(atomic.LoadInt64(&wsConnectionCount)))
	defer func() {
		atomic.AddInt64(&wsConnectionCount, -1)
		wsConnectionsActive.Set(float64(atomic.LoadInt64(&wsConnectionCount)))
	}()

	log.Printf("WebSocket connected user_id=%s", userID)

	ch, err := amqpConn.Channel()
	if err != nil {
		log.Printf("amqp channel error: %v", err)
		return
	}
	defer ch.Close()

	// Declare the exchange (idempotent).
	if err := ch.ExchangeDeclare(
		"onchain.alerts", "topic", true, false, false, false, nil,
	); err != nil {
		log.Printf("exchange declare error: %v", err)
		return
	}

	// Per-user queue: auto-deleted when the last consumer disconnects.
	q, err := ch.QueueDeclare(
		"alerts."+userID, false, true, false, false, nil,
	)
	if err != nil {
		log.Printf("queue declare error: %v", err)
		return
	}

	if err := ch.QueueBind(q.Name, "user."+userID, "onchain.alerts", false, nil); err != nil {
		log.Printf("queue bind error: %v", err)
		return
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Printf("consume error: %v", err)
		return
	}

	// Read goroutine: detect client disconnect.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg.Body); err != nil {
				log.Printf("websocket write error: %v", err)
				return
			}
			var alert AlertMessage
			if err := json.Unmarshal(msg.Body, &alert); err == nil {
				alertsDeliveredTotal.WithLabelValues(alert.ProtocolID).Inc()
			}
		case <-done:
			log.Printf("WebSocket disconnected user_id=%s", userID)
			return
		}
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[subscription] ")
	log.Println("Starting subscription service on :8084")

	ctx := context.Background()

	rdb = redis.NewClient(&redis.Options{
		Addr: getEnv("REDIS_ADDR", "redis:6379"),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	log.Printf("Redis connected at %s", getEnv("REDIS_ADDR", "redis:6379"))

	var err error
	amqpConn, err = amqp.Dial(getEnv("RABBITMQ_URL", "amqp://onchain:onchain@rabbitmq:5672/"))
	if err != nil {
		log.Fatalf("rabbitmq connection failed: %v", err)
	}
	defer amqpConn.Close()
	log.Printf("RabbitMQ connected at %s", getEnv("RABBITMQ_URL", "amqp://onchain:onchain@rabbitmq:5672/"))

	// Declare the exchange once at startup so it exists before any consumer binds to it.
	ch, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("amqp channel error: %v", err)
	}
	if err := ch.ExchangeDeclare(
		"onchain.alerts", "topic", true, false, false, false, nil,
	); err != nil {
		log.Fatalf("exchange declare error: %v", err)
	}
	ch.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/api/v1/subscriptions", subscriptionsHandler)
	mux.HandleFunc("/api/v1/subscriptions/", subscriptionsHandler)
	mux.HandleFunc("/ws", wsHandler)

	srv := &http.Server{
		Addr:         ":8084",
		Handler:      mux,
		ReadTimeout:  0, // disable for WebSocket connections
		WriteTimeout: 0,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
