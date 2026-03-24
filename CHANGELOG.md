# Changelog

All notable changes to this project are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions map to GitHub pull requests; the project does not use semver tags yet.

---

## [Unreleased]

---

## [0.9.0] - 2026-03-24 - PR #10: Docker build validation on every PR

### Fixed

- All six `build-and-push` CI jobs were gated with
  `if: github.ref == 'refs/heads/main'`, meaning a broken Dockerfile was only
  discovered after merge. The job now runs on every push and PR; only the push
  to GHCR remains conditional on `main`. The GHCR login step is also skipped
  on PRs so no credentials are required for a build-only run.

---

## [0.8.0] - 2026-03-23 - PR #8: Fix collector and analyzer Docker builds

### Fixed

- `COPY . .` was overwriting the `go.mod` that `go get` had just updated in
  an earlier layer, leaving `go mod tidy` with no new deps to resolve.
  Merged `go get`, `go mod tidy`, and `go build` into a single `RUN` after
  `COPY . .` in both Dockerfiles.

---

## [0.7.0] - 2026-03-23 - PR #7: Align Dockerfiles with CI dep resolution

### Fixed

- All Go service Docker builds were failing on `main` because `go.mod` files
  only declared the original Prometheus transitive deps. New direct deps added
  in previous PRs (Kafka, OTel, RabbitMQ, Redis) were resolved by `go get` in
  the CI lint step but never committed back to `go.mod`. The Docker build
  copied the stale `go.mod` and failed at compile time.
- Restructured `api` and `notifier` Dockerfiles to run `go get` for missing
  direct deps followed by `go mod tidy` before building.
- Renamed `dashboard/next.config.ts` to `next.config.mjs`. TypeScript config
  is only supported from Next.js 15+; the project targets Next.js 14.

---

## [0.6.0] - 2026-03-23 - PR #6: Next.js 14 App Router dashboard

### Added

- New `dashboard/` service: Next.js 14 App Router frontend on port 3001.
- Protocol health feed page polling `GET /api/v1/protocols` every 5 seconds,
  with colour-coded status badges (healthy / degraded / critical).
- Subscription management page: create, list, and delete alert subscriptions
  via the subscription REST API.
- Real-time alert panel connecting to the WebSocket endpoint
  (`ws://localhost:8084/ws?user_id=...`) and streaming incoming alerts live.
- Tailwind CSS for styling; no third-party component library.
- Multi-stage Dockerfile (node:20-alpine build, node:20-alpine runtime with
  Next.js standalone output).
- `CI - dashboard` GitHub Actions workflow: ESLint lint + `next build`.
- `dashboard` service added to `docker-compose.yml` on port 3001.

---

## [0.5.0] - 2026-03-23 - PR #5: RabbitMQ subscriptions and WebSocket delivery

### Added

- New `subscription` service (port 8084) with:
  - `POST /api/v1/subscriptions` - create a subscription (user + protocol + threshold).
  - `GET /api/v1/subscriptions/{user_id}` - list subscriptions for a user.
  - `DELETE /api/v1/subscriptions/{user_id}/{id}` - remove a subscription.
  - `GET /ws?user_id={user_id}` - WebSocket stream delivering real-time alerts.
- RabbitMQ `topic` exchange `onchain.alerts`; per-user auto-delete queues
  bound with routing key `user.{user_id}`.
- Redis subscription store with three-index key schema:
  `sub:{id}`, `user_subs:{user_id}`, `proto_subs:{protocol_id}`.
- `notifier` updated to look up matching subscriptions in Redis and publish
  `AlertMessage` to RabbitMQ when `score <= threshold`.
- Prometheus metrics on the subscription service: active subscriptions,
  active WebSocket connections, alerts delivered (labelled by protocol).
- ADR-014 (RabbitMQ topic exchange) and ADR-015 (Redis key schema) added to
  `docs/architecture/DECISIONS.md`.
- `CI - subscription` GitHub Actions workflow.
- RabbitMQ (port 5672 / 15672) and Redis (port 6379) added to
  `docker-compose.yml` with health checks.

### Fixed

- Removed stale `TestRandomScore_*` tests and renamed `sendNotification` to
  `sendSystemAlert` in `services/notifier/main_test.go` after the notifier
  was refactored in PR #4.

---

## [0.4.0] - 2026-03-23 - PR #4: Kafka event pipeline

### Added

- Apache Kafka (KRaft, single-node, bitnami/kafka:3.7) added to
  `docker-compose.yml` with a health check.
- `collector` now publishes a `DeFiEvent` (protocol ID, price, TVL, event
  type) to Kafka topic `onchain.events` after each simulation tick.
- `analyzer` replaced its random score generator with a Kafka consumer of
  `onchain.events`. Computes health scores from real price/TVL deviation:
  `score = int((priceNorm*0.5 + tvlNorm*0.5) * 100)` with a -10 penalty for
  liquidation events. Publishes `HealthEvent` to `onchain.health`.
- `notifier` updated to consume `onchain.health`; fires alerts on real scores
  instead of simulated data.
- `api` updated to consume `onchain.health`; serves live protocol state
  instead of hardcoded initial values.
- Consumer groups for independent offset tracking: `analyzer-group`,
  `notifier-group`, `api-group`.
- ADR-013 (Kafka, KRaft mode, `segmentio/kafka-go`, topic design) added.

---

## [0.3.0] - 2026-03-20 - PR #3: Unit tests and E2E suite

### Added

- Unit test suites for all four services (`collector`, `analyzer`, `notifier`,
  `api`) covering HTTP handlers, scoring logic, severity labels, and metrics.
- E2E smoke test in `e2e/e2e_test.go`: starts all services and asserts health,
  metrics, and protocol API endpoints respond correctly.
- `CI - e2e` GitHub Actions workflow running the E2E suite against live
  containers.

---

## [0.2.0] - 2026-03-20 - PR #2: CI stabilisation

### Fixed

- Pinned OTel to v1.28.0 and prevented `GOTOOLCHAIN` auto-switch in Go 1.23.
- Pinned `staticcheck` to v0.5.1 (the last release compatible with Go 1.23).
- Disabled markdownlint rules MD022 and MD049 broken by pre-existing doc
  formatting.
- Added `go get` + `go mod tidy` step to each CI workflow to resolve deps
  not declared in `go.mod` at the time of the initial commit.

---

## [0.1.0] - 2026-03-20 - PR #1 + initial release

### Added

- Initial release: 4 Go microservices (`collector`, `analyzer`, `notifier`,
  `api`) with simulated DeFi health monitoring.
- Docker Compose stack: 4 app services, Prometheus, Grafana, Jaeger, OTel
  Collector, Kong API gateway, Swagger UI.
- Multi-stage Dockerfiles (golang:1.23-alpine builder, alpine:3.19 runtime).
- Health score computation: `healthy` (>=70), `degraded` (>=40), `critical`
  (<40). Alert engine fires when score < 30.
- REST API: `GET /api/v1/protocols`, `GET /api/v1/protocols/{id}`.
- Prometheus metrics, Grafana dashboards, OpenTelemetry traces, Jaeger.
- Kong gateway (DB-less) with rate limiting, correlation ID header, and
  Prometheus scraping.
- OpenAPI 3.0 spec + Swagger UI.
- Terraform (GKE cluster + VPC + IAM) and Helm umbrella chart.
- 12 Architecture Decision Records in `docs/architecture/DECISIONS.md`.
- GitHub Actions CI per service: `go vet` + `staticcheck` + `go test -race` +
  Docker build/push to GHCR.
- PR quality checks: commitlint + markdownlint.

### Fixed

- Docker build/push CI: added `docker/setup-buildx-action` to enable GitHub
  Actions layer cache (PR #1).
