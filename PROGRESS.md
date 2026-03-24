# Progress

Current status of every planned feature and known issue.
Updated before each pull request.

---

## Architecture

```
collector → Kafka (onchain.events)
              → analyzer → Kafka (onchain.health)
                             → notifier → RabbitMQ (onchain.alerts) → subscription (WebSocket) → dashboard
                             → api (REST)
```

All services expose `/health` and `/metrics` (Prometheus).
Kong proxies the public API. OTel Collector forwards traces to Jaeger.

---

## Services

| Service | Port | Status | Notes |
|---------|------|--------|-------|
| collector | 8081 | done | Publishes DeFiEvent to Kafka every 2s |
| analyzer | 8082 | done | Computes health scores from price/TVL deviation |
| notifier | 8083 | done | Fires system alerts + routes user alerts via RabbitMQ |
| api | 8080 | done | Serves live protocol state from Kafka |
| subscription | 8084 | done | REST CRUD + WebSocket alert delivery |
| dashboard | 3001 | done | Next.js 14 health feed + subscription UI |

---

## Infrastructure

| Component | Status | Notes |
|-----------|--------|-------|
| Kafka (KRaft) | done | Topics: `onchain.events`, `onchain.health` |
| RabbitMQ | done | Exchange: `onchain.alerts` (topic), per-user auto-delete queues |
| Redis | done | Subscription store with 3-index key schema |
| Prometheus | done | Scrapes all services + Kong |
| Grafana | done | Dashboards provisioned automatically |
| Jaeger | done | Receives traces via OTel Collector |
| OTel Collector | done | Batch processor + logging exporter |
| Kong | done | Rate limiting, correlation ID, Prometheus plugin |
| Swagger UI | done | Served at port 8090 |

---

## CI/CD

| Workflow | Status | Trigger |
|----------|--------|---------|
| CI - collector | passing | push/PR on `services/collector/**` |
| CI - analyzer | passing | push/PR on `services/analyzer/**` |
| CI - notifier | passing | push/PR on `services/notifier/**` |
| CI - api | passing | push/PR on `services/api/**` |
| CI - subscription | passing | push/PR on `services/subscription/**` |
| CI - dashboard | passing | push/PR on `dashboard/**` |
| CI - infra | passing | push/PR on `infra/**` |
| CI - e2e | passing | push/PR on `e2e/**` |
| PR Checks | passing | all PRs (commitlint + markdownlint) |

Docker `build-and-push` now runs on every PR (build only) and pushes to GHCR
only on `main`. A broken Dockerfile will fail the PR before merge.

---

## Documentation

| Document | Status | Location |
|----------|--------|----------|
| Architecture overview | done | `docs/architecture/` |
| ADR-001 to ADR-015 | done | `docs/architecture/DECISIONS.md` |
| Developer onboarding | done | `docs/development/` |
| Local deployment guide | done | `docs/deployment/` |
| Operational runbooks | done | `docs/runbooks/` |
| OpenAPI 3.0 spec | done | `infra/api-spec/openapi.yaml` |
| Roadmap | done | `ROADMAP.md` |
| Changelog | done | `CHANGELOG.md` |
| Progress | done | `PROGRESS.md` (this file) |

---

## Backlog

Items not yet started, ordered roughly by priority.

### Near-term

- [x] Build (no push) Docker images on PRs so broken Dockerfiles are caught
      before merge.
- [ ] Replace the `go get` workaround in Dockerfiles with properly committed
      `go.mod` / `go.sum` files (requires Go installed in the dev environment
      or a CI step that commits updated module files).
- [ ] `golangci-lint` in CI (currently using standalone `go vet` + `staticcheck`).
- [ ] `govulncheck` in CI to surface known Go CVEs.
- [ ] Upgrade base Docker images to resolve known CVEs in `golang:1.23-alpine`
      and `alpine:3.19`.

### v1.2 - Real on-chain data

- [ ] Connect `collector` to a real RPC endpoint (`MOCK_MODE=false`).
- [ ] Support multiple chains (Ethereum mainnet, Arbitrum, Base).
- [ ] Expand protocol coverage (Curve, Balancer, MakerDAO).
- [ ] Kubernetes deployment via Helm on CI merge.

### v1.3 - Reliability and alerting

- [ ] PagerDuty / Slack webhook integration in `notifier`.
- [ ] Public Statuspage reflecting API health.
- [ ] Prometheus long-term storage (Thanos or VictoriaMetrics).

### v1.4 - Log aggregation

- [ ] Add Grafana Loki + Promtail to Docker Compose.
- [ ] Structured JSON logs via `log/slog` in all services.
- [ ] Correlate logs and traces via `trace_id` label.

### v1.5 - Supply chain security

- [ ] Sign container images with Cosign (Sigstore) on every release.
- [ ] Generate SBOM (CycloneDX) per service and attach to GitHub releases.
- [ ] Dependabot for Go module and GitHub Actions updates.

### v1.6 - Load testing

- [ ] k6 smoke test (5 VUs, 30s) on every merge to main.
- [ ] k6 soak test (50 VUs, 10min) weekly via scheduled workflow.
- [ ] Export k6 results to Prometheus for trend analysis.

### v2.1 - Real on-chain indexing

- [ ] The Graph Protocol subgraphs for indexed on-chain data.
- [ ] Chainlink Data Feeds as a price oracle source.
