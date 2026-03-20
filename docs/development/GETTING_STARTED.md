# Getting Started - OnChain Health Monitor

This guide walks you through setting up the project locally and verifying everything works.

---

## Prerequisites

| Tool | Minimum version | Install |
|------|----------------|---------|
| Go | 1.22 | https://go.dev/dl/ |
| Docker | 24.x | https://docs.docker.com/get-docker/ |
| Docker Compose | v2.x (`docker compose`) | Bundled with Docker Desktop; or `apt install docker-compose-plugin` |
| Make | any | Usually pre-installed; `brew install make` or `apt install make` |
| Git | 2.x | https://git-scm.com/ |

> **Note:** Docker Compose v2 is invoked as `docker compose` (no hyphen). The project's `docker-compose.yml` works with both v1 (`docker-compose`) and v2.

---

## Clone & First Run

```bash
git clone https://github.com/KaelSensei/OnChainHealthMonitor.git
cd OnChainHealthMonitor

# Build all 4 services and start the full stack (7 containers)
docker compose up --build
```

The first build downloads base images and compiles the Go binaries - expect ~2 minutes.  
On subsequent runs, `docker compose up` (without `--build`) starts in seconds.

You should see log lines like:
```
onchain_collector  | [collector] Starting collector service on :8081 (mock mode)
onchain_analyzer   | [analyzer]  Starting analyzer service on :8082
onchain_notifier   | [notifier]  Starting notifier service on :8083 (alert threshold: score < 30)
onchain_api        | [api]       Starting API service on :8080
```

---

## Service Ports

| Service | Container name | Port | Purpose |
|---------|---------------|------|---------|
| `api` | `onchain_api` | `8080` | Public REST API |
| `collector` | `onchain_collector` | `8081` | Mock event generator |
| `analyzer` | `onchain_analyzer` | `8082` | Health score computation |
| `notifier` | `onchain_notifier` | `8083` | Alert engine |
| Prometheus | `onchain_prometheus` | `9090` | Metrics scraping + query UI |
| Grafana | `onchain_grafana` | `3000` | Dashboards (admin / admin) |
| Jaeger | `onchain_jaeger` | `16686` | Distributed trace UI |
| OTel Collector | `onchain_otel_collector` | `4317` | OTLP gRPC receiver (used by services internally) |
| OTel Collector | `onchain_otel_collector` | `4318` | OTLP HTTP receiver |
| OTel Collector | `onchain_otel_collector` | `55679` | zpages debug interface |

---

## Verify All Services Are Running

Open a second terminal and run these health checks:

```bash
# API service
curl http://localhost:8080/health
# → {"status":"ok"}

# Collector
curl http://localhost:8081/health
# → {"status":"ok"}

# Analyzer
curl http://localhost:8082/health
# → {"status":"ok"}

# Notifier
curl http://localhost:8083/health
# → {"status":"ok"}
```

---

## API Endpoints

```bash
# List all monitored protocols with health scores
curl http://localhost:8080/api/v1/protocols

# Get a single protocol by ID
curl http://localhost:8080/api/v1/protocols/uniswap
curl http://localhost:8080/api/v1/protocols/aave
curl http://localhost:8080/api/v1/protocols/compound
```

Sample response for `GET /api/v1/protocols/uniswap`:
```json
{
  "id": "uniswap",
  "name": "Uniswap",
  "category": "DEX",
  "chain": "Ethereum",
  "health_score": 82,
  "status": "healthy",
  "tvl_usd": 4200000000,
  "price_usd": 6.52,
  "updated_at": "2026-03-18T21:00:00Z"
}
```

---

## Prometheus Metrics

Each service exposes raw Prometheus metrics at `/metrics`:

```bash
# Collector: price and TVL gauges per protocol
curl http://localhost:8081/metrics

# Analyzer: health score gauge per protocol
curl http://localhost:8082/metrics

# Notifier: total alerts fired counter
curl http://localhost:8083/metrics

# API: total HTTP requests counter
curl http://localhost:8080/metrics
```

Open the Prometheus query UI at **http://localhost:9090** to explore:

- `analyzer_health_score` - current health score per protocol (gauge)
- `collector_price_usd` - current price in USD per protocol
- `collector_tvl_usd` - total value locked in USD per protocol
- `notifier_alerts_total` - cumulative alert counter since startup
- `api_requests_total` - cumulative API request counter

---

## Grafana

Open **http://localhost:3000** in your browser.

- **Username:** `admin`  
- **Password:** `admin`

Prometheus is available as a data source at `http://prometheus:9090` (internal Docker network).  
Dashboard definitions are loaded from `observability/grafana/dashboards/` - currently a placeholder pending Phase 2.

---

## Jaeger (Distributed Tracing)

Open **http://localhost:16686** in your browser.

Jaeger receives traces from the OTel Collector, which batches and forwards spans exported by the Go services. The trace pipeline is:

```
Go service → otel-collector:4317 (gRPC) → jaeger:4317 → Jaeger UI
```

### Viewing Distributed Traces

1. Open **http://localhost:16686**
2. In the **Service** dropdown on the left, select a service:
   - `onchain-collector` - spans for each mock DeFi event generated (`generate_event`)
   - `onchain-analyzer` - spans for each health score computation (`analyze_protocol`)
3. Click **Find Traces** - each row is one complete trace
4. Click a trace to open the **span waterfall** timeline
5. Expand individual spans to see attributes (e.g. `protocol.id`, `health.score`, `event.type`)

Wide bars = slow operations. Nested bars = a call chain. Red spans = errors.

### OTel Collector Debug Interface (zpages)

If traces aren't appearing in Jaeger, check the OTel Collector's zpages debug UI:

```
http://localhost:55679
```

Navigate to `/debug/tracez` to see live span counts, pipeline stats, and any export errors.

---

## View Logs

```bash
# Follow logs for all services
docker compose logs -f

# Follow logs for a specific service
docker compose logs -f collector
docker compose logs -f analyzer
docker compose logs -f notifier
docker compose logs -f api

# Follow observability stack logs
docker compose logs -f prometheus
docker compose logs -f grafana
docker compose logs -f jaeger
```

The collector emits a JSON event every 2 seconds per protocol (3 protocols = 6 events per cycle):
```
[collector] {"timestamp":"2026-03-18T21:00:00Z","protocol_id":"uniswap",...}
```

The analyzer logs a score update every 3 seconds:
```
[analyzer] protocol=uniswap score=78 label=healthy
```

The notifier logs alert status every 5 seconds:
```
[notifier] 🔔 ALERT WARNING protocol="compound" score=22 ...
[notifier] ✅ All protocols healthy (no alerts)
```

---

## Run a Single Service Locally (without Docker)

Each service is a standalone Go binary with no external dependencies beyond the standard library.

```bash
# Run the collector locally
cd services/collector
go run .

# Run the analyzer locally
cd services/analyzer
go run .

# Run the notifier locally
cd services/notifier
go run .

# Run the API service locally
cd services/api
go run .
```

Each service starts on its default port. To avoid port conflicts with Docker Compose containers, stop the stack first with `docker compose down`.

---

## Stop the Stack

```bash
# Stop all containers (keeps volumes)
docker compose down

# Stop and remove all data volumes (full reset)
docker compose down -v
```

See `docs/deployment/LOCAL_SETUP.md` for more troubleshooting and reset options.
