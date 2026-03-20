# OnChain Health Monitor - Project Brief

## What it does

OnChain Health Monitor is a multi-service platform that tracks the health of decentralised finance (DeFi) protocols in real time. It ingests on-chain data (prices, TVL, protocol events), computes a health score for each protocol, fires alerts when scores degrade, and exposes everything through a public REST API.

The system runs in **mock mode by default** - a built-in data generator produces realistic synthetic events so the full pipeline works out of the box without an RPC key. Switching to live blockchain data is a single config change.

---

## Why this project

DeFi protocols handle billions of dollars in on-chain liquidity. When a protocol starts behaving abnormally - a sudden TVL drop, price deviation, or surge in liquidations - users and integrators need to know fast.

Existing on-chain monitoring solutions are either proprietary, chain-specific, or too expensive to self-host at small scale. This project provides an open, composable alternative: a pipeline you can run locally with `docker compose up`, extend with new protocols, and deploy to Kubernetes when you're ready.

The domain is also a natural fit for a multi-service architecture:
- Data ingestion (`collector`) is independent from analysis (`analyzer`)
- Alerting (`notifier`) is independent from the public API (`api`)
- Each service can be scaled, deployed, and observed independently

---

## Architecture principles

**Observable by default.** Every service exposes Prometheus metrics and OpenTelemetry traces from the first line of code. Dashboards and alerts are provisioned automatically - no manual setup required after `docker compose up`.

**Nothing clicked manually.** Infrastructure is defined in Terraform. Kubernetes deployments are managed by Helm. CI/CD is handled by GitHub Actions. The entire system - from cloud resources to Grafana dashboards - is reproducible from code.

**Config over code for real data.** The transition from mock to live blockchain data requires no refactoring. Set `MOCK_MODE=false` and provide `RPC_ENDPOINT` - the collector handles the rest.

**API-first.** The OpenAPI 3.0 spec is committed to the repo and served via Swagger UI through the Kong gateway. The contract is defined before the implementation.

---

## Stack decisions at a glance

| Theme | Choice | Rationale |
|---|---|---|
| Language | Go 1.22 | Fast startup, small binaries, excellent concurrency primitives |
| Metrics | Prometheus + Grafana | Self-hosted, pull-based, zero cost, industry standard |
| Tracing | OpenTelemetry + Jaeger | Vendor-neutral OTLP pipeline; swap backends via config |
| API Gateway | Kong (OSS) | Plugin ecosystem - rate limiting and auth without custom code |
| IaC | Terraform + Helm | Reproducible infra; reviewable in PRs like any other code |
| CI/CD | GitHub Actions | Native integration, path-based triggers for monorepo efficiency |

Full rationale for each decision is in [DECISIONS.md](./DECISIONS.md).

---

## Scope

**In scope (v1.0):**
- Mock data pipeline running end-to-end locally
- Full observability stack (Prometheus, Grafana, Jaeger, OTel Collector)
- Kong API gateway with rate limiting and Swagger UI
- GitHub Actions CI/CD with GHCR image push
- Terraform + Helm + Kubernetes infra skeleton (GKE-ready)
- Operational runbooks for each alert type

**Planned (v1.1+):**
- Real on-chain data via RPC endpoint
- Multi-chain support (Ethereum, Arbitrum, Base)
- Event-driven inter-service communication (NATS or Kafka)
- Historical health score API
- Public Statuspage

See [ROADMAP.md](../../ROADMAP.md) for the full list.
