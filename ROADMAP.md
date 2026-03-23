# OnChain Health Monitor - Roadmap

## Status: 🟢 v1.1 - Kafka event pipeline complete

---

## ✅ Milestone 1 - Core Services

- [x] 4 Go microservices (`collector`, `analyzer`, `notifier`, `api`)
- [x] Docker Compose (4 app services + Prometheus + Grafana + Jaeger + OTel Collector)
- [x] Multi-stage Dockerfiles (golang:1.22-alpine → alpine:3.19)
- [x] Health score computation with status label (`healthy` / `degraded` / `critical`)
- [x] Alert engine (fires when score < 30)
- [x] REST API (`GET /api/v1/protocols`, `GET /api/v1/protocols/{id}`)
- [x] `GET /health` and `GET /metrics` on all 4 services
- [x] Mock event generator (price drift, TVL, DeFi events every 2s)

## ✅ Milestone 2 - Observability

- [x] Prometheus metrics (counters, histograms, gauges) via `promhttp`
- [x] Grafana dashboards (latency, error rate, throughput per service)
- [x] OpenTelemetry instrumentation (spans + trace context propagation)
- [x] Jaeger trace collection (end-to-end: api → analyzer → collector)
- [x] OTel Collector as pipeline intermediary (batch processor, logging exporter)
- [x] Alerting rules (SLO-based: error rate > 1%, p99 latency > 500ms)
- [x] Grafana provisioning (datasources auto-configured on `docker compose up`)

## ✅ Milestone 3 - API Gateway

- [x] Kong gateway in Docker Compose (DB-less mode)
- [x] Rate limiting plugin
- [x] Correlation-ID header (`X-Request-ID`) via request-transformer plugin
- [x] Kong Prometheus plugin (scrape target added to `prometheus.yml`)
- [x] OpenAPI 3.0 spec (`infra/api-spec/openapi.yaml`)
- [x] Swagger UI served via Kong

## ✅ Milestone 4 - CI/CD

- [x] GitHub Actions: `go vet` + `staticcheck` + `go test -race` + Docker build per service
- [x] Path-based triggers (only rebuilds the changed service)
- [x] Push to GHCR on merge to `main`
- [x] Semantic version releases on `v*.*.*` tags (matrix build across all 4 services)
- [x] PR quality checks: commitlint + markdownlint
- [ ] Deploy to Kubernetes via Helm on merge to main _(planned v1.1)_

## ✅ Milestone 5 - Infrastructure as Code

- [x] Terraform: GKE cluster + VPC + IAM (`infra/terraform/`)
- [x] Helm umbrella chart + per-service subcharts (`infra/helm/`)
- [x] Kubernetes manifests: Deployments, Services, HPA, ServiceMonitors (`infra/k8s/`)
- [ ] Statuspage integration _(planned v1.2)_

## ✅ Milestone 6 - Documentation

- [x] 12 Architecture Decision Records (ADRs) including ADR-013 for Kafka
- [x] Technical architecture doc
- [x] Developer onboarding guide
- [x] Contributing guide (branch naming, commit conventions, code style)
- [x] Local deployment guide
- [x] Operational runbooks for each alert type
- [x] Full API reference (OpenAPI 3.0)
- [ ] Grafana dashboard screenshot in README _(planned v1.2)_
- [ ] Public Statuspage URL _(planned v1.3)_

## ✅ Milestone 7 - Kafka Event Pipeline

- [x] Apache Kafka (KRaft, single-node) added to Docker Compose
- [x] `collector` publishes `DeFiEvent` to Kafka topic `onchain.events`
- [x] `analyzer` consumes `onchain.events`, computes real health scores from price/TVL deviation, publishes `HealthEvent` to `onchain.health`
- [x] `notifier` consumes `onchain.health` and fires alerts on real scores (no more simulated data)
- [x] `api` consumes `onchain.health` and serves live scores (no more hardcoded state)
- [x] Consumer groups for independent offset tracking: `analyzer-group`, `notifier-group`, `api-group`
- [x] ADR-013 documenting the Kafka decision, topic design, and trade-offs

---

## 🔜 v1.2 - Real On-Chain Data

- [ ] Connect `collector` to a real RPC endpoint (`MOCK_MODE=false`)
- [ ] Support multiple chains (Ethereum mainnet, Arbitrum, Base)
- [ ] Expand protocol coverage (Curve, Balancer, MakerDAO)
- [ ] Kubernetes deployment via Helm on CI merge
- [ ] Grafana dashboard screenshot in README

## 🔜 v1.3 - Reliability & Alerting

- [ ] PagerDuty / Slack webhook integration in `notifier`
- [ ] Public Statuspage reflecting API health
- [ ] Grafana SLO dashboards (error budget burn rate)
- [ ] Prometheus long-term storage (Thanos sidecar or VictoriaMetrics)

## 🔜 v1.4 - Log Aggregation (Loki)

Complete the three pillars of observability - metrics (Prometheus) and traces (Jaeger) are in; logs are the missing piece.

- [ ] Add Grafana Loki + Promtail to the Docker Compose stack
- [ ] Ship structured JSON logs from all 4 services via `log/slog` (Go 1.21+)
- [ ] Configure Loki as a Grafana datasource (alongside Prometheus and Jaeger)
- [ ] Add log-based alerting rules (error bursts, panic detection)
- [ ] Correlate logs ↔ traces via `trace_id` label in Loki

## 🔜 v1.5 - Supply Chain Security

- [ ] Sign container images with Cosign (Sigstore) on every release
- [ ] Generate SBOM (CycloneDX) per service in CI and attach to GitHub releases
- [ ] Add Dependabot for automated Go module and GitHub Actions dependency updates
- [ ] Add `golangci-lint` to CI (replaces standalone `staticcheck`)
- [ ] Add `govulncheck` to CI to scan for known Go vulnerabilities

## 🔜 v1.6 - Load Testing & SLO Validation

- [ ] Add k6 load test suite (`tests/k6/`) targeting the Kong-proxied API
- [ ] Smoke test (5 VUs, 30s) on every merge to main
- [ ] Soak test (50 VUs, 10min) weekly via scheduled GitHub Actions workflow
- [ ] Export k6 results to Prometheus remote write → Grafana dashboard for trend analysis
- [ ] Define error budget burn rate alerts based on load test SLO targets

## 🔜 v2.0 - User Subscriptions (RabbitMQ)

Kafka handles the data pipeline. RabbitMQ handles per-user routing: each user subscribes to specific protocols and thresholds, and receives targeted alerts via WebSocket.

- [ ] RabbitMQ added to Docker Compose alongside Kafka (topic exchanges for per-user routing)
- [ ] Subscription model: users define `{protocol, threshold}` pairs stored in a database
- [ ] `notifier` publishes to RabbitMQ when a health score crosses a user subscription threshold
- [ ] WebSocket server bridges RabbitMQ queues to connected browser clients
- [ ] Next.js dashboard with real-time protocol health feed and subscription management UI
- [ ] Historical health score API (`GET /api/v1/protocols/{id}/history`)
- [ ] Cross-service trace context propagation via Kafka and RabbitMQ message headers

## 🔜 v2.1 - Real On-Chain Indexing

- [ ] Integrate The Graph Protocol subgraphs for indexed on-chain data (no RPC rate limits)
- [ ] Support Chainlink Data Feeds as a price oracle source
- [ ] Multi-chain support: Ethereum mainnet, Arbitrum, Base
- [ ] Alchemy / QuickNode webhook receiver for real-time protocol events
