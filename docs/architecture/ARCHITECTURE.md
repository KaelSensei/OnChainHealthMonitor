# Architecture - OnChain Health Monitor

## System Overview

OnChain Health Monitor is a monorepo housing four Go microservices that together form a continuous on-chain DeFi monitoring pipeline. The system ingests mock DeFi events (or real blockchain data via RPC), computes protocol health scores, triggers alerts on degradation, and exposes the results through a REST API. An observability stack (Prometheus, Grafana, Jaeger) runs alongside the application services.

The design philosophy: **functionally simple, architecturally serious.** The domain logic is intentionally shallow so that the infrastructure patterns - distributed tracing, metrics, containerisation, API gateway, IaC - are the real focus.

---

## Service Interaction Diagram

```mermaid
graph TD
    subgraph External["External Traffic"]
        Client([Client / Browser])
    end

    subgraph Gateway["API Gateway"]
        Kong["Kong\n:8000 proxy\n:8001 admin"]
    end

    subgraph Broker["Message Brokers"]
        Kafka["Kafka (KRaft)\n:9092\nonchain.events\nonchain.health"]
        RabbitMQ["RabbitMQ\n:5672 / :15672\nexchange: onchain.alerts\n(topic)"]
    end

    subgraph DataStore["Data Store"]
        Redis["Redis\n:6379"]
    end

    subgraph Services["Go Microservices"]
        Collector["collector\n:8081\nDeFi event producer"]
        Analyzer["analyzer\n:8082\nHealth score computation"]
        Notifier["notifier\n:8083\nAlert engine"]
        API["api\n:8080\nREST API"]
        Subscription["subscription\n:8084\nUser subscriptions + WebSocket"]
    end

    subgraph Observability["Observability Stack"]
        Prometheus["Prometheus\n:9090"]
        Grafana["Grafana\n:3000"]
        OtelCollector["OTel Collector\n:4317"]
        Jaeger["Jaeger\n:16686"]
    end

    subgraph Docs["API Docs"]
        Swagger["Swagger UI\n:8090"]
    end

    Client -->|"HTTP :8000"| Kong
    Kong -->|"proxy"| API
    Kong -->|"/swagger"| Swagger

    Collector -->|"publish DeFiEvent\nonchain.events"| Kafka
    Kafka -->|"consume DeFiEvent\nanalyzer-group"| Analyzer
    Analyzer -->|"publish HealthEvent\nonchain.health"| Kafka
    Kafka -->|"consume HealthEvent\nnotifier-group"| Notifier
    Kafka -->|"consume HealthEvent\napi-group"| API
    Notifier -->|"lookup subscriptions\nproto_subs:{protocol_id}"| Redis
    Notifier -->|"publish AlertMessage\nrouting key user.{user_id}"| RabbitMQ
    RabbitMQ -->|"deliver to queue\nalerts.{user_id}"| Subscription
    Subscription -->|"subscription CRUD\nsub:{id}, user_subs:{user_id}"| Redis

    Collector -->|"OTLP gRPC"| OtelCollector
    Analyzer -->|"OTLP gRPC"| OtelCollector
    OtelCollector -->|"OTLP"| Jaeger

    Prometheus -->|"scrape /metrics"| Collector
    Prometheus -->|"scrape /metrics"| Analyzer
    Prometheus -->|"scrape /metrics"| Notifier
    Prometheus -->|"scrape /metrics"| API
    Prometheus -->|"scrape /metrics"| Kong
    Grafana -->|"PromQL"| Prometheus
    Grafana -->|"traces"| Jaeger
```

---

## Data Flow

```mermaid
sequenceDiagram
    participant C as collector
    participant KE as Kafka<br/>onchain.events
    participant A as analyzer
    participant KH as Kafka<br/>onchain.health
    participant N as notifier
    participant R as Redis
    participant RMQ as RabbitMQ<br/>onchain.alerts
    participant SUB as subscription
    participant WS as WebSocket Client
    participant API as api
    participant K as Kong
    participant P as Prometheus
    participant OC as OTel Collector
    participant J as Jaeger
    participant G as Grafana
    participant CL as Client

    loop Every 2 seconds (per protocol)
        C->>C: Generate DeFiEvent (price, TVL, event_type)
        C->>OC: Export span: generate_event
        C->>KE: Publish DeFiEvent (key=protocol_id)
    end

    loop On every message (analyzer-group)
        KE->>A: Consume DeFiEvent
        A->>A: Compute health score from price/TVL deviation
        A->>OC: Export span: analyze_protocol
        A->>KH: Publish HealthEvent (score, label, price, tvl)
    end

    loop On every message (notifier-group)
        KH->>N: Consume HealthEvent
        alt score < 30
            N->>N: Log ALERT (WARNING or CRITICAL)
        end
        N->>R: SMEMBERS proto_subs:{protocol_id}
        R-->>N: matching subscription IDs
        loop For each matching subscription
            N->>R: GET sub:{id} (fetch user_id, threshold)
            alt score <= threshold
                N->>RMQ: Publish AlertMessage (routing key user.{user_id})
                RMQ->>SUB: Deliver to queue alerts.{user_id}
                SUB->>WS: Push alert via WebSocket
            end
        end
    end

    loop On every message (api-group)
        KH->>API: Consume HealthEvent
        API->>API: Update in-memory protocol state
    end

    loop Every 15 seconds
        P->>C: Scrape /metrics
        P->>A: Scrape /metrics
        P->>N: Scrape /metrics
        P->>API: Scrape /metrics
    end

    CL->>K: GET /api/v1/protocols
    K->>API: Proxy request (rate limit check)
    API->>K: Response (live scores from last HealthEvent)
    K->>CL: Response + X-Request-ID header

    OC->>J: Forward traces (batch)
    G->>P: Query PromQL
    G->>J: Query traces
```

---

## Services

### `collector` - Event Ingestion

| Property | Value |
|----------|-------|
| Port | `8081` |
| Role | Ingests DeFi protocol data and publishes events to Kafka |
| Inputs | None in mock mode; RPC endpoint in real mode |
| Outputs | `DeFiEvent` messages on `onchain.events`; metrics on `/metrics` |

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check → `{"status":"ok"}` |
| `GET` | `/metrics` | Prometheus gauges: `collector_price_usd`, `collector_tvl_usd` |

**Mock mode behaviour:**
- Runs a goroutine (`emitLoop`) that ticks every 2 seconds
- For each of 3 protocols (Uniswap, Aave, Compound), generates a `DeFiEvent` with:
  - Price: random walk ±2% per tick, clamped to ±20% of baseline
  - TVL: random walk ±1% per tick, clamped to ±20% of baseline
  - Event type: one of `price_update`, `tvl_change`, `swap`, `liquidation`, `deposit`
  - Volume: 1-5% of current TVL
- Each `DeFiEvent` is published to Kafka topic `onchain.events` with `protocol_id` as the key

**Baseline values:**

| Protocol | Price (USD) | TVL (USD) |
|----------|------------|-----------|
| Uniswap | $6.50 | $4.2B |
| Aave | $95.00 | $6.1B |
| Compound | $52.00 | $2.3B |

---

### `analyzer` - Health Score Computation

| Property | Value |
|----------|-------|
| Port | `8082` |
| Role | Consumes `DeFiEvent` messages, computes health scores, publishes `HealthEvent` messages |
| Inputs | `DeFiEvent` messages from Kafka topic `onchain.events` (consumer group `analyzer-group`) |
| Outputs | `HealthEvent` messages on Kafka topic `onchain.health`; metrics on `/metrics` |

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check → `{"status":"ok"}` |
| `GET` | `/metrics` | Prometheus gauge: `analyzer_health_score{protocol="..."}` |

**Score computation:**

The health score maps price and TVL deviation from their baselines to a 0-100 scale:

- Score 50 when price and TVL are exactly at baseline
- Score 100 when both are at +20% of baseline
- Score 0 when both are at -20% of baseline
- Liquidation events apply an additional -10 penalty

```
priceNorm = clamp((price - baseline*0.8) / (baseline*0.4), 0, 1)
tvlNorm   = clamp((tvl   - baseline*0.8) / (baseline*0.4), 0, 1)
score     = int((priceNorm*0.5 + tvlNorm*0.5) * 100)
```

**Score labels:**

| Score range | Label |
|-------------|-------|
| 70-100 | `healthy` |
| 40-69 | `degraded` |
| 0-39 | `critical` |

---

### `notifier` - Alert Engine

| Property | Value |
|----------|-------|
| Port | `8083` |
| Role | Consumes `HealthEvent` messages and fires alerts when scores drop below threshold |
| Inputs | `HealthEvent` messages from Kafka topic `onchain.health` (consumer group `notifier-group`) |
| Outputs | Alert log to stdout; `notifier_alerts_total` counter on `/metrics` |

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check → `{"status":"ok"}` |
| `GET` | `/metrics` | Prometheus counter: `notifier_alerts_total` |

**Alert logic:**
- Fires on every `HealthEvent` message where `score < 30`
- Severity levels: `WARNING` (score 20-29), `CRITICAL` (score < 20)
- A webhook channel is simulated for `CRITICAL` severity (score < 20)
- Future: real integrations with PagerDuty, Slack, or Grafana Alerting webhooks

---

### `api` - Public REST API

| Property | Value |
|----------|-------|
| Port | `8080` |
| Role | Exposes protocol health data to external consumers |
| Inputs | `HealthEvent` messages from Kafka topic `onchain.health` (consumer group `api-group`) |
| Outputs | JSON REST responses |

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check → `{"status":"ok"}` |
| `GET` | `/metrics` | Prometheus counter: `api_requests_total` |
| `GET` | `/api/v1/protocols` | List all protocols with health scores |
| `GET` | `/api/v1/protocols/{id}` | Get a single protocol by ID |

**Protocol response schema:**

```json
{
  "id": "string",
  "name": "string",
  "category": "string (DEX | Lending)",
  "chain": "string",
  "health_score": "integer (0–100)",
  "status": "string (healthy | degraded | critical)",
  "tvl_usd": "float",
  "price_usd": "float",
  "updated_at": "ISO 8601 timestamp"
}
```

**List response schema:**
```json
{
  "protocols": [...],
  "total": "integer"
}
```

**Error response:**
```json
{"error": "protocol \"unknown\" not found"}
```
HTTP 404 for unknown protocol IDs.

---

### `subscription` - User Subscriptions and Real-Time Alerts

| Property | Value |
|----------|-------|
| Port | `8084` |
| Role | Manages user subscriptions and delivers real-time alerts via WebSocket |
| Inputs | REST requests for CRUD; RabbitMQ queue (onchain.alerts exchange, routing key `user.{user_id}`) |
| Outputs | JSON REST responses; WebSocket alert messages |

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/subscriptions` | Create a subscription (`{user_id, protocol_id, threshold}`) |
| `GET` | `/api/v1/subscriptions/{user_id}` | List subscriptions for a user |
| `DELETE` | `/api/v1/subscriptions/{user_id}/{id}` | Delete a subscription |
| `GET` | `/ws?user_id={user_id}` | WebSocket alert stream |

**Subscription schema:**

```json
{
  "id": "uuid",
  "user_id": "string",
  "protocol_id": "string",
  "threshold": "integer (1-100)",
  "created_at": "ISO 8601 timestamp"
}
```

---

## Observability Stack

### Metrics - Prometheus + Grafana

```
Each service exposes GET /metrics (Prometheus text format, content-type: text/plain; version=0.0.4)
          │
          │ scrape every 10s (per-service jobs in prometheus.yml)
          ▼
    Prometheus :9090
          │
          │ data source
          ▼
    Grafana :3000
          │
          │ dashboards (Phase 2: latency, error rate, health scores, alert counts)
          ▼
    Grafana Alerting (Phase 2: SLO rules → webhook → notification channels)
```

**Current metrics exposed:**

| Service | Metric | Type | Description |
|---------|--------|------|-------------|
| collector | `collector_price_usd{protocol}` | Gauge | Current price in USD |
| collector | `collector_tvl_usd{protocol}` | Gauge | Current TVL in USD |
| analyzer | `analyzer_health_score{protocol}` | Gauge | Current health score (0–100) |
| notifier | `notifier_alerts_total` | Counter | Total alerts fired since startup |
| api | `api_requests_total` | Counter | Total HTTP requests handled |

**Prometheus scrape config:** `observability/prometheus/prometheus.yml`  
Scrape interval: 10s per service, 15s global.

### Traces - OpenTelemetry + Jaeger

#### Distributed Tracing

All four Go services are instrumented with the OpenTelemetry Go SDK. Each service initialises an OTLP gRPC exporter at startup, pointing at the OTel Collector (`otel-collector:4317`). If the collector is unreachable, the service logs a warning and continues running without tracing (graceful degradation).

**Trace pipeline:**

```
Go service (OTLP gRPC) → otel-collector:4317 → batch processor → jaeger:4317 → Jaeger UI (:16686)
```

The OTel Collector is configured in `observability/otel/otel-collector-config.yaml`:
- **Receiver:** OTLP (gRPC on `:4317`, HTTP on `:4318`)
- **Processor:** batch (1s timeout, 1024 spans/batch)
- **Exporter:** `otlp/jaeger` (forwards to Jaeger) + `logging` (prints spans to collector logs for debugging)
- **zpages debug UI:** `http://localhost:55679`

**Spans currently instrumented:**

| Service | Span | Attributes |
|---------|------|-----------|
| `collector` | `generate_event` | `protocol.id`, `event.type`, `price.usd`, `tvl.usd` |
| `analyzer` | `analyze_protocol` | `protocol.id`, `health.score`, `health.label` |

Jaeger is auto-provisioned as a Grafana datasource, so you can correlate traces directly from Grafana dashboards.

See `docs/development/TRACING_GUIDE.md` for how to search traces in the Jaeger UI and how to add spans to new service code.

---

## Mock Mode

The `collector` service runs in **mock mode by default** - it generates realistic but synthetic DeFi data internally without requiring any external connections. This means:

- The full 4-service pipeline runs end-to-end from `docker compose up`
- Grafana dashboards and alerts are populated with live-looking data immediately
- No API keys, no blockchain RPC quotas, no rate-limiting issues during development

**Switching to real data** (planned Phase 2):
```bash
MOCK_MODE=false
RPC_ENDPOINT=https://mainnet.infura.io/v3/<KEY>
```
This is a configuration change, not a code change.

---

## Docker Compose Network Layout

All containers share a default bridge network created by Docker Compose. Service discovery uses Docker's built-in DNS - containers reference each other by service name (e.g., `prometheus` scrapes `http://collector:8081/metrics`).

```
Docker bridge network: onchain_network
  ├── kafka          (onchain_kafka)           :9092 → host:9092
  ├── rabbitmq       (onchain_rabbitmq)        :5672 / :15672 → host
  ├── redis          (onchain_redis)           :6379 → host
  ├── collector      (onchain_collector)       :8081 → host:8081
  ├── analyzer       (onchain_analyzer)        :8082 → host:8082
  ├── notifier       (onchain_notifier)        :8083 → host:8083
  ├── api            (onchain_api)             :8080 → host:8080
  ├── subscription   (onchain_subscription)   :8084 → host:8084
  ├── prometheus     (onchain_prometheus)      :9090 → host:9090
  ├── grafana        (onchain_grafana)         :3000 → host:3000
  ├── jaeger         (onchain_jaeger)          :16686 (UI) / :4317 (gRPC fallback) → host
  └── otel-collector (onchain_otel_collector)  :4317 (gRPC) / :4318 (HTTP) / :55679 (zpages) → host
```

**Note:** Services export traces to `otel-collector:4317` (internal Docker DNS). The otel-collector container is not exposed on host port 4317 - that port is mapped to Jaeger instead (for direct OTLP fallback). The zpages debug interface (`localhost:55679`) is the primary way to verify the collector is receiving spans.

**Volumes:**
- `grafana_data` - persists Grafana state (dashboards, users) across restarts

**Startup dependencies (Docker Compose `depends_on`):**
- `collector`, `analyzer`, `notifier`, `api` all depend on `kafka` with `condition: service_healthy`
- Kafka uses a 30s `start_period` healthcheck because the broker takes ~30s to be ready
- `grafana` depends on `prometheus`

---

## Environment Variables Reference

### `collector`

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `kafka:9092` | Comma-separated list of Kafka broker addresses |
| `MOCK_MODE` | `true` | Set `false` to connect to a real RPC endpoint |
| `RPC_ENDPOINT` | _(none)_ | Blockchain RPC URL (used when `MOCK_MODE=false`) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel-collector:4317` | OTLP gRPC endpoint for trace export |

### `analyzer`

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `kafka:9092` | Comma-separated list of Kafka broker addresses |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel-collector:4317` | OTLP gRPC endpoint for trace export |

### `notifier`

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `kafka:9092` | Comma-separated list of Kafka broker addresses |
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `RABBITMQ_URL` | `amqp://onchain:onchain@rabbitmq:5672/` | AMQP connection URL |

### `api`

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `kafka:9092` | Comma-separated list of Kafka broker addresses |

### `subscription`

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `RABBITMQ_URL` | `amqp://onchain:onchain@rabbitmq:5672/` | AMQP connection URL |

> All four pipeline services read `KAFKA_BROKERS` from the environment. The value is set to `kafka:9092` in `docker-compose.yml` and should point to the broker list for your environment.

---

## Repository Structure

```
OnChainHealthMonitor/
├── services/
│   ├── collector/          # Mock DeFi event generator
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── Dockerfile
│   ├── analyzer/           # Health score computation
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── Dockerfile
│   ├── notifier/           # Alert engine
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── Dockerfile
│   └── api/                # Public REST API
│       ├── main.go
│       ├── go.mod
│       └── Dockerfile
├── infra/
│   ├── terraform/          # GCP/GKE infrastructure (Phase 5)
│   ├── helm/               # Helm charts per service (Phase 5)
│   └── k8s/                # Raw Kubernetes manifests (Phase 5)
├── observability/
│   ├── prometheus/
│   │   └── prometheus.yml  # Scrape configuration
│   ├── grafana/
│   │   └── dashboards/     # Dashboard JSON definitions (Phase 2)
│   ├── otel/               # OpenTelemetry collector config (Phase 2)
│   └── jaeger/             # Jaeger configuration (Phase 2)
├── docs/
│   ├── architecture/
│   │   ├── PROJECT_BRIEF.md
│   │   ├── ARCHITECTURE.md  ← this file
│   │   └── DECISIONS.md
│   ├── deployment/
│   │   └── LOCAL_SETUP.md
│   └── development/
│       ├── GETTING_STARTED.md
│       └── CONTRIBUTING.md
├── .github/
│   └── workflows/           # GitHub Actions pipelines (Phase 4)
├── docker-compose.yml
├── ROADMAP.md
└── README.md
```
