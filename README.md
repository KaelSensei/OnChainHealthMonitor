<p align="center">
  <img src="assets/banner.svg" alt="OnChain Health Monitor" width="100%"/>
</p>

<p align="center">
  <strong>Functionally simple. Architecturally serious.</strong>
</p>

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/github/license/KaelSensei/OnChainHealthMonitor)](LICENSE)

[![CI - api](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-api.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-api.yml)
[![CI - collector](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-collector.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-collector.yml)
[![CI - analyzer](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-analyzer.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-analyzer.yml)
[![CI - notifier](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-notifier.yml/badge.svg)](https://github.com/KaelSensei/OnChainHealthMonitor/actions/workflows/ci-notifier.yml)

---

## Architecture

```mermaid
graph TD
    subgraph External["🌐 External Traffic"]
        Client([Client / Browser])
    end

    subgraph Gateway["🦍 API Gateway"]
        Kong["Kong\n:8000 proxy\n:8001 admin"]
    end

    subgraph Services["⚙️ Go Microservices"]
        Collector["collector\n:8081\nDeFi event ingestion"]
        Analyzer["analyzer\n:8082\nHealth scoring"]
        Notifier["notifier\n:8083\nAlerting"]
        API["api\n:8080\nREST API"]
    end

    subgraph Observability["📊 Observability Stack"]
        Prometheus["Prometheus\n:9090"]
        Grafana["Grafana\n:3000"]
        OtelCollector["OTel Collector\n:4317"]
        Jaeger["Jaeger\n:16686"]
    end

    subgraph Docs["📋 API Docs"]
        Swagger["Swagger UI\n:8090"]
    end

    Client -->|"HTTP :8000"| Kong
    Kong -->|"proxy"| API
    Kong -->|"/swagger"| Swagger

    Collector -->|"events"| Analyzer
    Analyzer -->|"scores"| Notifier
    API -->|"reads scores"| Analyzer

    Collector -->|"OTLP gRPC"| OtelCollector
    Analyzer -->|"OTLP gRPC"| OtelCollector
    Notifier -->|"OTLP gRPC"| OtelCollector
    API -->|"OTLP gRPC"| OtelCollector
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
    participant A as analyzer
    participant N as notifier
    participant API as api
    participant K as Kong
    participant P as Prometheus
    participant OC as OTel Collector
    participant J as Jaeger
    participant G as Grafana
    participant CL as Client

    loop Every 2 seconds
        C->>C: Ingest DeFi event<br/>(price, TVL, protocol_event)
        C->>OC: Export trace span (generate_event)
        C->>A: Emit event data
        A->>A: Compute health score (0–100)
        A->>OC: Export trace span (analyze_protocol)
        alt score < 30
            A->>N: Trigger alert
            N->>N: Log 🔔 ALERT
            N->>OC: Export trace span (send_notification)
        end
    end

    loop Every 15 seconds
        P->>C: Scrape /metrics
        P->>A: Scrape /metrics
        P->>N: Scrape /metrics
        P->>API: Scrape /metrics
    end

    CL->>K: GET /api/v1/protocols
    K->>API: Proxy request (rate limit check)
    API->>API: Return protocol list with health scores
    API->>OC: Export trace span (GET /api/v1/protocols)
    API->>K: Response
    K->>CL: Response + X-Request-ID header

    OC->>J: Forward traces (batch)
    G->>P: Query PromQL
    G->>J: Query traces
```

---

## Services

| Service     | Port | Role                                                              |
|-------------|------|-------------------------------------------------------------------|
| `collector` | 8081 | Ingests on-chain data (mock mode: emits DeFi events every 2s)    |
| `analyzer`  | 8082 | Processes events, computes health scores (0–100) per protocol     |
| `notifier`  | 8083 | Fires alerts when any protocol score drops below threshold (30)   |
| `api`       | 8080 | REST API exposing protocol health data to external consumers      |

All services expose:
- `GET /health` → `{"status":"ok"}` for liveness checks
- `GET /metrics` → Prometheus text format

---

## Stack

| Theme                    | Tool                        | Why                                                      |
|--------------------------|-----------------------------|----------------------------------------------------------|
| Language                 | Go 1.22                     | Fast, minimal stdlib, perfect for microservices          |
| Containers               | Docker + Docker Compose     | Reproducible local environment, mirrors prod topology    |
| Observability: Metrics   | Prometheus + Grafana        | Industry standard; scrape model fits pull-based services |
| Observability: Tracing   | OpenTelemetry + OTel Collector + Jaeger | OTLP gRPC pipeline: services → collector → Jaeger UI |
| CI/CD                    | GitHub Actions              | Native to GitHub, path-based triggers for monorepos      |
| Reliability / Alerting   | Grafana Alerting            | Unified alerting with SLO-based rules, no extra infra    |
| API Gateway              | Kong (open-source)          | Plugin ecosystem (rate limit, auth, logging) on OSS      |
| Infra as Code            | Terraform                   | Declarative, provider-agnostic, auditable history        |
| Kubernetes packaging     | Helm                        | Templated manifests, per-environment value overrides     |
| Cloud                    | GCP / GKE (or k3s locally)  | k3s for zero-cost dev; GKE for real deployment           |

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

# Start the full stack (builds all 4 services + spins up Prometheus, Grafana, Jaeger)
docker-compose up --build

# In another terminal, verify services
curl http://localhost:8080/health           # API
curl http://localhost:8080/api/v1/protocols # Protocol list
curl http://localhost:8081/health           # Collector
curl http://localhost:8082/health           # Analyzer
curl http://localhost:8083/health           # Notifier

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
│   └── api/           # Public REST API
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
│   ├── architecture/  # ARCHITECTURE.md + ADRs
│   ├── deployment/    # Infrastructure & CI/CD guides
│   └── development/   # Onboarding guide
├── .github/
│   └── workflows/     # GitHub Actions pipelines
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
        Docker["docker build\n+ push GHCR"]
    end

    subgraph Infra["Infra validation"]
        Compose["docker compose config"]
        Kong["deck file validate\nkong.yaml"]
        OApi["redocly lint\nopenapi.yaml"]
    end

    subgraph Release["Release on tag v*.*.*"]
        Matrix["matrix build\ncollector / analyzer\nnotifier / api"]
        GHCR[("ghcr.io/kaelsensei\n/onchainhealthmonitor")]
    end

    Commit --> CommitLint
    Commit --> MDLint
    Commit --> Vet --> Static --> Test --> Build --> Docker
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
