<p align="center">
  <img src="assets/banner.svg" alt="OnChain Health Monitor" width="100%"/>
</p>

<p align="center">
  <strong>Functionally simple. Architecturally serious.</strong>
</p>

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/github/license/KaelSensei/OnChainHealthMonitor)](LICENSE)

[![CI - api](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-api.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-api.yml)
[![CI - collector](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-collector.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-collector.yml)
[![CI - analyzer](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-analyzer.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-analyzer.yml)
[![CI - notifier](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-notifier.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-notifier.yml)
[![CI - subscription](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-subscription.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-subscription.yml)
[![CI - dashboard](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-dashboard.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-dashboard.yml)

---

## Architecture

```mermaid
graph TD
    subgraph External["External Traffic"]
        Browser([Browser])
    end

    subgraph Frontend["Frontend"]
        Dashboard["dashboard\n:3001\nNext.js App Router"]
    end

    subgraph Gateway["API Gateway"]
        Kong["Kong\n:8000 proxy\n:8001 admin"]
    end

    subgraph Broker["Message Brokers"]
        Kafka["Kafka KRaft\n:9092\nonchain.events\nonchain.health"]
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

    Browser -->|"HTTP :3001"| Dashboard
    Dashboard -->|"server-side proxy"| API
    Dashboard -->|"server-side proxy"| Subscription
    Browser -->|"WebSocket :8084"| Subscription
    Browser -->|"HTTP :8000"| Kong
    Kong -->|"proxy"| API
    Kong -->|"/swagger"| Swagger

    Collector -->|"publish DeFiEvent"| Kafka
    Kafka -->|"consume DeFiEvent\nanalyzer-group"| Analyzer
    Analyzer -->|"publish HealthEvent"| Kafka
    Kafka -->|"consume HealthEvent\nnotifier-group"| Notifier
    Kafka -->|"consume HealthEvent\napi-group"| API
    Notifier -->|"lookup subscriptions"| Redis
    Notifier -->|"route per-user alert"| RabbitMQ
    RabbitMQ -->|"deliver alert"| Subscription
    Subscription <-->|"subscription CRUD"| Redis

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
    participant KE as Kafka onchain.events
    participant A as analyzer
    participant KH as Kafka onchain.health
    participant N as notifier
    participant R as Redis
    participant RMQ as RabbitMQ onchain.alerts
    participant SUB as subscription
    participant WS as WebSocket Client
    participant DASH as dashboard
    participant API as api
    participant K as Kong
    participant P as Prometheus
    participant CL as Browser

    loop Every 2s per protocol
        C->>C: Generate DeFiEvent (price, TVL, event_type)
        C->>KE: Publish DeFiEvent (key=protocol_id)
    end

    loop On every DeFiEvent (analyzer-group)
        KE->>A: Consume DeFiEvent
        A->>A: Compute score from price/TVL deviation
        A->>KH: Publish HealthEvent (score, label, price, tvl)
    end

    loop On every HealthEvent (notifier-group)
        KH->>N: Consume HealthEvent
        alt score < 30
            N->>N: Log ALERT (WARNING or CRITICAL)
        end
        N->>R: Lookup proto_subs:{protocol_id}
        R-->>N: Matching subscription IDs
        loop For each matching subscription
            N->>R: GET sub:{id}
            alt score <= threshold
                N->>RMQ: Publish AlertMessage (routing key user.{user_id})
                RMQ->>SUB: Deliver to queue alerts.{user_id}
                SUB->>WS: Push alert via WebSocket
            end
        end
    end

    loop On every HealthEvent (api-group)
        KH->>API: Consume HealthEvent
        API->>API: Update in-memory protocol state
    end

    loop Every 15s
        P->>C: Scrape /metrics
        P->>A: Scrape /metrics
        P->>N: Scrape /metrics
        P->>API: Scrape /metrics
    end

    CL->>DASH: GET / (initial page load)
    DASH->>API: Server-side fetch /api/v1/protocols
    API-->>DASH: Protocol list (live scores)
    DASH-->>CL: Rendered HTML with protocol cards

    loop Every 5s (client-side polling)
        CL->>DASH: GET /api/protocols (Next.js route handler)
        DASH->>API: Proxy request
        API-->>DASH: Updated scores
        DASH-->>CL: JSON response
    end

    CL->>K: GET /api/v1/protocols (direct API access)
    K->>API: Proxy (rate limit check)
    API->>K: Live scores from last HealthEvent
    K->>CL: Response + X-Request-ID
```

---

## Services

| Service        | Port | Role                                                                        |
|----------------|------|-----------------------------------------------------------------------------|
| `collector`    | 8081 | Generates DeFi events (mock mode) and publishes to `onchain.events`         |
| `analyzer`     | 8082 | Consumes `onchain.events`, computes health scores, publishes `onchain.health` |
| `notifier`     | 8083 | Consumes `onchain.health`, fires alerts when score drops below 30           |
| `api`          | 8080 | Consumes `onchain.health`, serves live protocol data via REST               |
| `subscription` | 8084 | Manages user subscriptions (CRUD + Redis), delivers real-time alerts via WebSocket (RabbitMQ) |
| `dashboard`    | 3001 | Next.js App Router UI: protocol health feed, subscription management, real-time alert stream |

All services expose:
- `GET /health` → `{"status":"ok"}` for liveness checks
- `GET /metrics` → Prometheus text format

---

## Stack

| Theme                      | Tool                        | Why                                                      |
|----------------------------|-----------------------------|----------------------------------------------------------|
| Language (backend)         | Go 1.23                     | Fast, minimal stdlib, perfect for microservices          |
| Language (frontend)        | TypeScript + Next.js 14     | App Router enables SSR for initial data; client components for real-time updates; API routes as BFF proxy |
| Message Broker             | Apache Kafka (KRaft)        | High-throughput event streaming; decouples all services; replay support |
| User notification routing  | RabbitMQ                    | Topic exchange pattern routes alerts to specific users; auto-delete queues per connected client |
| Subscription storage       | Redis                       | Fast set-based lookup by protocol; per-user subscription index |
| Containers                 | Docker + Docker Compose     | Reproducible local environment, mirrors prod topology    |
| Observability: Metrics     | Prometheus + Grafana        | Industry standard; scrape model fits pull-based services |
| Observability: Tracing     | OpenTelemetry + OTel Collector + Jaeger | OTLP gRPC pipeline: services -> collector -> Jaeger UI |
| CI/CD                      | GitHub Actions + Husky      | GHA for remote; Husky for local commit-msg + pre-commit hooks |
| Reliability / Alerting     | Grafana Alerting            | Unified alerting with SLO-based rules, no extra infra    |
| API Gateway                | Kong (open-source)          | Plugin ecosystem (rate limit, auth, logging) on OSS      |
| Infra as Code              | Terraform                   | Declarative, provider-agnostic, auditable history        |
| Kubernetes packaging       | Helm                        | Templated manifests, per-environment value overrides     |
| Cloud                      | GCP / GKE (or k3s locally)  | k3s for zero-cost dev; GKE for real deployment           |

---

## Production Deployment (GKE)

See [Infrastructure Guide](docs/deployment/INFRASTRUCTURE_GUIDE.md) for full details.

**Quick summary:**
1. Provision GKE: `cd infra/terraform && terraform apply`
2. Deploy with Helm: `helm install onchain-health-monitor ./infra/helm/onchain-health-monitor -n onchain-health-monitor --create-namespace`
3. Check status: `kubectl get pods -n onchain-health-monitor`

---

## Quick Start

```bash
# Clone
git clone https://github.com/KaelSensei/OnChainHealthMonitor.git
cd OnChainHealthMonitor

# Install dev tooling (Husky hooks, commitlint, lint-staged)
npm install

# Start the full stack
docker-compose up --build

# In another terminal, verify services
curl http://localhost:8080/health           # API
curl http://localhost:8080/api/v1/protocols # Protocol list
curl http://localhost:8081/health           # Collector
curl http://localhost:8082/health           # Analyzer
curl http://localhost:8083/health           # Notifier
curl http://localhost:8084/health           # Subscription

# Dashboard
open http://localhost:3001   # Next.js dashboard

# Observability UIs
open http://localhost:9090   # Prometheus
open http://localhost:3000   # Grafana  (admin / admin)
open http://localhost:16686  # Jaeger - distributed trace UI
open http://localhost:55679  # OTel Collector zpages - debug pipeline stats
open http://localhost:8090/swagger  # Swagger UI - interactive API docs
open http://localhost:8000/swagger  # Swagger UI via Kong gateway
```

### API Examples

```bash
# List all monitored protocols
curl http://localhost:8080/api/v1/protocols

# Get a single protocol
curl http://localhost:8080/api/v1/protocols/uniswap
curl http://localhost:8080/api/v1/protocols/aave
curl http://localhost:8080/api/v1/protocols/compound
```

---

## Mock Mode vs Real Data

The `collector` service runs in **mock mode by default** - it generates realistic synthetic DeFi data without requiring external connections, so the full pipeline works out of the box.

Switching to real on-chain data is a config change, not a code change:

```bash
MOCK_MODE=false
RPC_ENDPOINT=https://mainnet.infura.io/v3/<YOUR_KEY>
```

---

## Decision Log

### Why Go?

Minimal external dependencies, single static binary, excellent stdlib HTTP server - ideal for writing observable microservices without framework overhead. Each service compiles to a ~5MB binary.

### Why Prometheus + Grafana over Datadog?

Open-source, self-hosted, zero cost. Pull-based scraping matches the `/metrics` endpoints natively. Grafana Alerting provides SLO-based rules without a separate tool.

### Why Jaeger?

Native OTLP receiver, lightweight all-in-one Docker image, clean UI. OpenTelemetry SDK is vendor-neutral - swapping Jaeger for Honeycomb or Tempo is a config change.

### Why GitHub Actions?

The repo lives on GitHub. Native integration means no extra webhook setup, and path-based triggers (`on: push: paths:`) enable proper monorepo CI - only the changed service gets rebuilt.

### Why Kong?

Open-source, plugin-based, battle-tested at scale. Rate limiting, authentication, and request logging are single-line plugin configurations - no custom middleware code needed.

### Why Terraform?

Declarative, provider-agnostic, and produces an auditable state file. Infrastructure changes are reviewed in PRs the same way code changes are - nothing is clicked manually.

### Why Helm?

Kubernetes manifests need per-environment value overrides (image tag, replica count, resource limits). Helm templates are the standard way to manage that across staging/production without duplicating YAML.

---

## Project Structure

```
OnChainHealthMonitor/
├── services/
│   ├── collector/     # DeFi event ingestion + HTTP server
│   ├── analyzer/      # Health score computation
│   ├── notifier/      # Alert engine
│   ├── api/           # Public REST API
│   └── subscription/  # Subscription CRUD + WebSocket alert delivery
├── dashboard/         # Next.js 14 frontend (protocol health + subscriptions + 55 Vitest tests)
├── e2e/               # End-to-end smoke test suite
├── infra/
│   ├── terraform/     # GCP/GKE infrastructure as code
│   ├── helm/          # Helm charts per service
│   └── k8s/           # Raw Kubernetes manifests
├── observability/
│   ├── prometheus/    # Scrape configuration
│   ├── grafana/       # Dashboard definitions
│   ├── otel/          # OpenTelemetry collector config
│   └── jaeger/        # Jaeger configuration
├── docs/
│   ├── architecture/  # ARCHITECTURE.md + 16 ADRs
│   ├── deployment/    # Infrastructure & CI/CD guides
│   └── development/   # Onboarding, contributing, tracing guides
├── scripts/           # Developer utility scripts
├── .github/
│   └── workflows/     # GitHub Actions pipelines (10 workflows)
├── .husky/            # Git hooks: commit-msg + pre-commit
├── package.json       # Root dev tooling: Husky, commitlint, lint-staged
├── docker-compose.yml
└── README.md
```

---

## CI/CD Pipeline

```mermaid
graph LR
    subgraph PR["Pull Request"]
        Commit[/"git push"/]
        CommitLint["commitlint\n✓ conventional commits"]
        MDLint["markdownlint\n✓ docs quality"]
    end

    subgraph CI["CI per service (path-triggered)"]
        Vet["go vet ./..."]
        Static["staticcheck"]
        Test["go test -race"]
        Build["go build"]
        NpmLint["npm run lint"]
        NpmBuild["npm run build"]
        NpmTest["npm run test"]
        Docker["docker build (all branches)\n+ push GHCR (main only)"]
    end

    subgraph Infra["Infra validation"]
        Compose["docker compose config"]
        Kong["deck file validate\nkong.yaml"]
        OApi["redocly lint\nopenapi.yaml"]
    end

    subgraph Release["Release on tag v*.*.*"]
        Matrix["matrix build\ncollector / analyzer\nnotifier / api\nsubscription / dashboard"]
        GHCR[("ghcr.io/kaelsensei\n/onchainhealthmonitor")]
    end

    Commit --> CommitLint
    Commit --> MDLint
    Commit --> Vet --> Static --> Test --> Build --> Docker
    Commit --> NpmLint --> NpmBuild --> NpmTest --> Docker
    Commit --> Compose
    Commit --> Kong
    Commit --> OApi
    Docker --> GHCR
    Matrix --> GHCR
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [ROADMAP.md](./ROADMAP.md) | Project roadmap - completed milestones and upcoming work |
| [Architecture](./docs/architecture/ARCHITECTURE.md) | System overview, service responsibilities, data flow, environment variables |
| [Architecture Decisions (ADRs)](./docs/architecture/DECISIONS.md) | Why each tool was chosen: Go, Prometheus, Grafana, OTel, Jaeger, Kong, Terraform, Helm |
| [Getting Started](./docs/development/GETTING_STARTED.md) | Developer onboarding: prerequisites, first run, service ports, curl examples |
| [Tracing Guide](./docs/development/TRACING_GUIDE.md) | How to view traces in Jaeger, use zpages, and add spans to service code |
| [Local Setup](./docs/deployment/LOCAL_SETUP.md) | Full docker-compose setup, troubleshooting, reset procedures |
| [Contributing](./docs/development/CONTRIBUTING.md) | Branch naming, conventional commits, how to add a service or metric, code style |
| [CI/CD Guide](./docs/deployment/CI_CD_GUIDE.md) | GitHub Actions pipeline: workflows, path triggers, GHCR, releases, PR checks |
| [Infrastructure Guide](./docs/deployment/INFRASTRUCTURE_GUIDE.md) | Terraform, Helm, and Kubernetes - provisioning GKE and deploying all services |
| [Project Brief](./docs/architecture/PROJECT_BRIEF.md) | Project scope, motivation, and tool selection rationale |
| [Runbooks](./docs/runbooks/README.md) | Operational runbooks for each Grafana alert - what to do when an alert fires |

---

## License

MIT © KaelSensei
