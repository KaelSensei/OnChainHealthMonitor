# Architecture Decision Records

This document captures the key architectural decisions made for the OnChain Health Monitor project. Each record explains the context, the decision, and the trade-offs accepted.

---

## ADR-001: Go as the service language

**Date:** 2026-03-18  
**Status:** Accepted

**Context:**
We needed a language for four microservices that would produce small Docker images, start quickly, handle concurrent HTTP serving cleanly, and not require external framework dependencies in Phase 1.

**Decision:**  
Use Go 1.22 for all four services. Each service uses only the standard library (`net/http`, `encoding/json`, `sync`, `math/rand`, `time`, `log`).

**Consequences:**
- ✅ Single static binary per service (~5MB); multi-stage Docker build produces a minimal `alpine:3.19` image
- ✅ Built-in HTTP server is production-grade; no framework needed for this scope
- ✅ `goroutine` + `sync.RWMutex` pattern cleanly separates event loops from HTTP handlers
- ✅ `CGO_ENABLED=0` build is trivially cross-compiled for Linux containers from any host OS
- ⚠️ No dependency management complexity in Phase 1 (each `go.mod` is standalone); will need to add real deps (OpenTelemetry SDK, promhttp) in Phase 2

---

## ADR-002: Prometheus for metrics collection

**Date:** 2026-03-18  
**Status:** Accepted

**Context:**  
The project needs to instrument all four services with metrics to populate Grafana dashboards and trigger alerts. Options considered: Prometheus (self-hosted, pull-based), Datadog (SaaS, push-based), StatsD (push-based, no query language).

**Decision:**  
Use Prometheus with a pull-based scrape model. Each service exposes `GET /metrics` in Prometheus text format. Prometheus scrapes all four services on a 10s interval.

**Consequences:**
- ✅ Self-hosted, zero cost - matches the open-source-first ethos
- ✅ Pull model means services don't need to know where Prometheus is; scrape targets are configured centrally in `prometheus.yml`
- ✅ Prometheus text format is trivial to implement manually (Phase 1) and with `promhttp` (Phase 2)
- ✅ `PromQL` is the de-facto industry standard for metrics querying - ubiquitous in production monitoring stacks
- ✅ Native Grafana integration: Prometheus is the default Grafana data source
- ⚠️ No long-term storage (7-day TSDB retention in current config); would need Thanos or Cortex for production

---

## ADR-003: Grafana for dashboards and alerting

**Date:** 2026-03-18  
**Status:** Accepted

**Context:**  
We need dashboards to visualise metrics (latency, error rate, health scores) and an alerting mechanism to fire on SLO breaches. Options: Grafana (self-hosted, OSS), Datadog (SaaS), New Relic (SaaS), custom dashboards.

**Decision:**  
Use Grafana OSS 10.4 for both dashboards and alerting. Grafana Alerting (formerly Grafana unified alerting) evaluates PromQL rules and fires to notification channels.

**Consequences:**
- ✅ Self-hosted, open-source, no per-seat cost
- ✅ Grafana Alerting replaces the need for a separate Alertmanager for simple use cases
- ✅ Dashboard JSON can be version-controlled in `observability/grafana/dashboards/` and loaded as read-only mounts - "infrastructure as code" for dashboards
- ✅ SLO-based alerting rules (error rate > 1%, p99 > 500ms) are production-grade reliability patterns
- ⚠️ Grafana persistent state is stored in a Docker volume (`grafana_data`); dashboards defined outside the volume need provisioning config in Phase 2

---

## ADR-004: OpenTelemetry for distributed tracing instrumentation

**Date:** 2026-03-18  
**Status:** Accepted

**Context:**  
In a multi-service system, understanding request latency across service boundaries requires distributed tracing. OpenTelemetry is the CNCF standard SDK for emitting traces, metrics, and logs. Alternatives: direct Jaeger SDK (vendor lock-in), Zipkin SDK (less ecosystem momentum), no tracing (insufficient for production observability requirements).

**Decision:**  
Instrument all four services using the OpenTelemetry Go SDK (`go.opentelemetry.io/otel`). Export spans via OTLP gRPC to the OTel Collector. Each service bootstraps a `TracerProvider` at startup with a `BatchSpanProcessor` backed by an `otlptracegrpc` exporter. If the collector endpoint is unreachable at startup, the service logs a warning and continues running without tracing (graceful degradation).

**Implemented spans:**
- `collector`: `generate_event` - instruments each mock DeFi event, with attributes `protocol.id`, `event.type`, `price.usd`, `tvl.usd`
- `analyzer`: `analyze_protocol` - instruments each health score computation, with attributes `protocol.id`, `health.score`, `health.label`
- `notifier` and `api`: instrumentation planned (HTTP handler spans, alert evaluation spans)

**Consequences:**
- ✅ Vendor-neutral: switching from Jaeger to Honeycomb or Grafana Tempo is a one-line config change (OTLP endpoint)
- ✅ OpenTelemetry is the standard answer in any distributed systems interview
- ✅ Graceful degradation means tracing failures never take down the service
- ✅ Span attributes (protocol name, health score) enable filtering by DeFi protocol in the Jaeger UI
- ⚠️ Adds non-trivial dependency to `go.mod` for each instrumented service
- ⚠️ Cross-service trace context propagation (via `traceparent` HTTP header) is planned for Phase 2 when services communicate directly

---

## ADR-005: Jaeger as the trace backend

**Date:** 2026-03-18  
**Status:** Accepted

**Context:**  
We need a backend to receive, store, and visualise distributed traces. Options: Jaeger (CNCF, self-hosted), Zipkin (older, less OTLP-native), Grafana Tempo (would require Loki stack), Honeycomb (SaaS, paid).

**Decision:**  
Use `jaegertracing/all-in-one:1.57` as a single Docker container. It accepts OTLP gRPC on port 4317 (for direct fallback), and serves the UI on port 16686. `COLLECTOR_OTLP_ENABLED=true` is set in the Compose file. In normal operation, Jaeger receives traces from the OTel Collector (not directly from services - see ADR-011). Jaeger is also auto-provisioned as a Grafana datasource via `observability/grafana/provisioning/datasources/jaeger.yaml`, enabling trace lookups directly from Grafana dashboards.

**Consequences:**
- ✅ Single container - no separate collector/query/agent deployment for local dev
- ✅ Native OTLP receiver - no Jaeger-specific SDK needed; standard `go.opentelemetry.io/otel/exporters/otlp/otlptrace` works out of the box
- ✅ Clean trace UI with service dependency graph and span waterfall
- ✅ Auto-provisioned Grafana datasource means zero manual configuration after `docker compose up`
- ⚠️ `all-in-one` stores traces in memory; data is lost on container restart - acceptable for dev, not for production
- ⚠️ For production, Jaeger's distributed deployment (Cassandra/Elasticsearch backend) would be required

---

## ADR-006: GitHub Actions for CI/CD

**Date:** 2026-03-18  
**Status:** Accepted

**Context:**  
The project needs a CI/CD system to lint, test, build Docker images, and push to a container registry on every merge to main. The repo is hosted on GitHub. Options: GitHub Actions (native), GitLab CI (requires GitLab), CircleCI (SaaS), Jenkins (self-hosted complexity).

**Decision:**  
Use GitHub Actions with seven workflow files. Each of the four services gets its own workflow (`ci-api.yml`, `ci-collector.yml`, `ci-analyzer.yml`, `ci-notifier.yml`) with path-based triggers so only the changed service is rebuilt. Three additional workflows handle infrastructure validation (`ci-infra.yml`), releases (`release.yml`), and PR quality checks (`pr-checks.yml`).

**Key implementation details:**

- **Path-based triggers** - Each service workflow watches only its own directory (`services/<name>/**`) plus its own workflow file. Changing `services/collector/` runs only `ci-collector.yml`; the other three are untouched.
- **Linting** - `go vet ./...` for correctness, then `staticcheck ./...` (installed at runtime via `go install`) for deeper analysis. `staticcheck` was chosen over `golangci-lint` to keep the dependency footprint minimal and avoid aggregator complexity.
- **Testing** - `go test ./... -v -race -coverprofile=coverage.out` - race detector is always on.
- **Image registry** - GHCR (GitHub Container Registry) at `ghcr.io/kaelsensei/onchainhealthmonitor/<service>`. Authentication uses the automatic `GITHUB_TOKEN` - no manual secret to rotate, no external registry account needed.
- **Image tagging** - `sha-<short-commit>` and `latest` on every push to `main`; semantic version tags (`v1.2.3`, `v1.2`, `v1`) on git tag pushes via `release.yml`.
- **PR quality** - `pr-checks.yml` runs commitlint (enforces Conventional Commits via `.commitlintrc.json`) and markdownlint (enforces style via `.markdownlint.json`) on every pull request.
- **Build gate** - `build-and-push` job has `needs: lint-and-test` and only runs on `main` branch pushes, not PRs. A failing test never produces an image.

**Consequences:**
- ✅ Zero external setup - no webhook configuration, secrets sync, or separate CI account
- ✅ Free tier sufficient for this project (2,000 min/month on public repos)
- ✅ Path-based triggers is the correct monorepo CI pattern - unmodified services are never rebuilt
- ✅ GHCR is the natural push target - same org, free for public repos, no extra credentials
- ✅ `staticcheck` catches real bugs (incorrect API usage, unreachable code) with zero config
- ✅ commitlint enforces clean conventional commit history - directly readable by `git log`
- ✅ `docker/metadata-action` generates both `sha-` and semver tags automatically - no bash scripting
- ⚠️ No deployment step yet - `helm upgrade --install` on merge to main is planned for Phase 5
- ⚠️ GitHub Actions YAML can grow complex for monorepos; reusable workflows (`workflow_call`) are the future direction if jobs need further deduplication

---

## ADR-007: Kong as the API gateway

**Date:** 2026-03-18  
**Status:** Accepted (implementation pending Phase 3)

**Context:**  
External traffic to the `api` service needs rate limiting, authentication, and request logging without writing custom middleware. Options: Kong (OSS, plugin-based), AWS API Gateway (cloud-specific, costs money), Envoy (complex config for this scale), nginx (manual config, no plugin ecosystem).

**Decision:**  
Use Kong OSS. All external traffic is routed through Kong before reaching the `api` service. Rate limiting, key-auth, and request-transformer are configured as Kong plugins.

**Consequences:**
- ✅ Plugin ecosystem means zero custom middleware code for auth, rate limiting, logging
- ✅ Kong is battle-tested at scale; its presence on a CV is recognisable
- ✅ Declarative configuration (`kong.yml`) can be version-controlled
- ✅ Swagger UI served via Kong's Swagger plugin without a separate container
- ⚠️ Kong adds ~200MB Docker image overhead and a separate database (or DB-less mode) - DB-less mode with `kong.yml` is the planned approach for simplicity

---

## ADR-008: Terraform for infrastructure as code

**Date:** 2026-03-18  
**Status:** Accepted

**Context:**  
The Kubernetes cluster and supporting infrastructure (networking, IAM, container registry) must be reproducible, reviewable, and not manually provisioned. Options: Terraform (provider-agnostic, HCL), Pulumi (code-based, more complex), CDK (AWS-specific), manual console clicks (not acceptable).

**Decision:**  
Use Terraform to define GKE cluster, VPC, and service accounts. For local development, k3s replaces GKE at zero cost. The same Helm charts are used in both environments.

**Key implementation details:**

- **Module structure** - `infra/terraform/` uses two reusable modules: `modules/networking/` (VPC + subnet `10.0.0.0/16` in `europe-west1` with Pod/Service secondary ranges) and `modules/gke/` (cluster with workload identity, shielded nodes, autoscaling node pool of 1–5 × `e2-medium`)
- **Provider** - `hashicorp/google` targeting GCP `europe-west1`; requires Terraform ≥ 1.7.0
- **Vars file** - `terraform.tfvars.example` provides a ready-to-copy template; operators copy to `terraform.tfvars` and set `project_id`
- **State backend stub** - GCS backend block is stubbed in `main.tf` comments; local state is used for solo development, remote GCS state is the documented upgrade path for team use
- **Outputs** - `terraform output` emits the exact `gcloud container clusters get-credentials` command to configure kubectl post-apply

**Consequences:**
- ✅ "Nothing clicked manually" - Terraform PRs are infra reviews, making changes auditable and reversible
- ✅ Provider-agnostic: swapping GKE for EKS or AKS is a provider change, not a rewrite
- ✅ Terraform state file provides an auditable record of infrastructure changes
- ✅ Module separation means networking and GKE can be reviewed and versioned independently
- ⚠️ Terraform state must be stored remotely (GCS bucket) to support team collaboration; local state is acceptable for single-developer setups
- ⚠️ GKE incurs real cost if left running; k3s is the zero-cost alternative for demos

---

## ADR-009: Kubernetes + Helm for service deployment

**Date:** 2026-03-18  
**Status:** Accepted

**Context:**  
The four microservices need to be deployed in a way that supports per-environment configuration (image tag, replica count, resource limits), rolling updates, and horizontal scaling. Options: plain Kubernetes YAML (no templating), Helm (templated, per-environment values), Kustomize (overlay-based, less popular for new projects), ArgoCD (CD layer, not a packaging tool).

**Decision:**  
Use Helm charts per service via an **umbrella chart** pattern. `infra/helm/onchain-health-monitor/` is the parent chart; it declares the four per-service subcharts (api, collector, analyzer, notifier) as dependencies. Raw Kubernetes manifests in `infra/k8s/` handle cluster-level concerns (namespace, ServiceMonitors, Prometheus ConfigMap).

**Key implementation details:**

- **Umbrella chart** - `helm dep update onchain-health-monitor` resolves subcharts; a single `helm install` deploys all four services atomically
- **Per-service subcharts** - each contains Deployment, Service (ClusterIP), HPA, and ConfigMap templates
- **Image source** - all images pulled from `ghcr.io/kaelsensei/onchainhealthmonitor/<service>:latest` (same GHCR registry as CI/CD pipeline)
- **HPA on api** - scales between 2 and 10 replicas based on 70% CPU target; other services have HPA stubs ready to enable
- **Namespace** - `onchain-health-monitor`; isolated from system workloads, scoped for RBAC
- **ServiceMonitors** - four `ServiceMonitor` CRDs in `infra/k8s/` enable automatic Prometheus scraping via the Prometheus Operator without hard-coding pod IPs
- **Prometheus ConfigMap** - `infra/k8s/prometheus-config.yaml` configures scrape intervals (10s) inside the cluster

**Consequences:**
- ✅ Per-environment value files mean the same chart is deployed to staging and production with different image tags and replica counts
- ✅ `helm diff` enables reviewing infrastructure changes before applying, similar to `terraform plan`
- ✅ Helm is the standard packaging mechanism for Kubernetes - the default choice in the cloud-native ecosystem
- ✅ Umbrella chart pattern means one command deploys the entire application - easy for CI/CD integration
- ✅ ServiceMonitors decouple observability config from pod scheduling - scrape targets auto-update as pods restart
- ⚠️ Helm templates can become verbose; keeping charts minimal (Deployment + Service + HPA per service) avoids complexity
- ⚠️ Helm v3 eliminates Tiller (the server-side component from v2), removing the main historical security concern
- ⚠️ Prometheus Operator must be installed separately for ServiceMonitors to take effect (`prometheus-community/kube-prometheus-stack`)

---

## ADR-010: Grafana Alerting for SLO-based alerts

**Date:** 2026-03-18  
**Status:** Accepted (implementation pending Phase 2)

**Context:**  
The project needs SLO-based alerting (e.g., error rate > 1%, p99 latency > 500ms) that fires to a notification channel when breached. Options: Prometheus Alertmanager (separate deployment), Grafana Alerting (built into Grafana), PagerDuty (SaaS, requires paid tier for full features), custom polling code.

**Decision:**  
Use Grafana Alerting (unified alerting, enabled by default in Grafana 9+). Alert rules are defined in Grafana against PromQL queries. Notification channels are configured per alert group (email, Slack, or webhook).

**Consequences:**
- ✅ No additional container needed - Grafana already runs in the stack
- ✅ Alert rules live alongside the dashboards that visualise the same metrics
- ✅ Grafana Alerting supports silences, inhibitions, and contact points - production-ready feature set
- ✅ SLO-based rules encode reliability expectations in a reviewable, version-controlled form
- ⚠️ Alert rule definitions must be provisioned as code (Grafana provisioning YAML) to avoid "click-ops" - this is planned for Phase 2
- ⚠️ For true multi-team alerting at scale, Alertmanager with routing trees is more flexible; Grafana Alerting is the right call for a single-team project

---

## ADR-011: OTel Collector as trace pipeline intermediary

**Date:** 2026-03-18  
**Status:** Accepted

**Context:**  
Services could export traces directly to Jaeger using the OTLP gRPC exporter pointed at `jaeger:4317`. However, this tightly couples the services to the Jaeger backend - changing the storage backend would require updating environment variables in every service. Additionally, direct export provides no batching, filtering, or pipeline visibility.

**Decision:**  
Use the OpenTelemetry Collector (`otel/opentelemetry-collector-contrib:0.100.0`) as an intermediary pipeline. Services export to `otel-collector:4317`. The collector runs a `receivers → processors → exporters` pipeline: OTLP receiver → batch processor → Jaeger exporter + logging exporter. Collector config lives in `observability/otel/otel-collector-config.yaml`.

**Consequences:**
- ✅ Swapping Jaeger for Grafana Tempo or another OTLP-compatible backend is a one-line change in the collector config - no service code or env var changes needed
- ✅ The batch processor reduces network chatter (1s timeout, 1024 spans/batch)
- ✅ The logging exporter prints raw span data to the collector's stdout - invaluable for debugging when traces don't appear in Jaeger
- ✅ The zpages debug interface (`http://localhost:55679`) provides live pipeline stats without restarting the stack
- ⚠️ Adds one more container to the Compose file and one more dependency to reason about
- ⚠️ If the collector itself is unhealthy, traces are lost even if Jaeger is fine - mitigated by `restart: unless-stopped` in docker-compose

---

## ADR-013: Apache Kafka as the event streaming backbone

**Date:** 2026-03-23
**Status:** Accepted

**Context:**
In Phase 1, services communicated via in-process simulated state. Each service generated its own random data independently; there was no real data flow between the collector, analyzer, notifier, and API. This was intentional for a minimal proof of concept, but it meant the system was not actually monitoring anything. Moving toward real on-chain data requires a durable, high-throughput transport layer that can handle the volume of blockchain events and decouple producers from consumers.

**Decision:**
Introduce Apache Kafka (KRaft mode, single-node) as the event streaming backbone. Two topics are used:

- `onchain.events` - the collector publishes one `DeFiEvent` per protocol per tick (currently every 2 seconds, ~1.5 events/sec in mock mode, orders of magnitude more with real RPC data)
- `onchain.health` - the analyzer publishes one `HealthEvent` per consumed `DeFiEvent`, containing the computed health score, label, price, and TVL

Consumer groups:

| Consumer | Topic | Group ID |
|---|---|---|
| analyzer | onchain.events | analyzer-group |
| notifier | onchain.health | notifier-group |
| api | onchain.health | api-group |

The Go client used is `github.com/segmentio/kafka-go` (pure Go, no CGO dependency).

KRaft mode (no ZooKeeper) is used because ZooKeeper was deprecated in Kafka 3.x and removed in Kafka 4.0. A single-node KRaft cluster is sufficient for development and staging; a multi-broker setup with replication is expected for production.

**Consequences:**
- ✅ Services are now truly decoupled: the collector, analyzer, notifier, and API each operate independently and communicate only through Kafka topics
- ✅ Consumer group offsets allow each consumer to read at its own pace; if the notifier restarts it resumes from its last committed offset and processes no missed health events
- ✅ Kafka's append-only log enables replay: adding a new consumer (for example an ML anomaly detector or an archiver) requires no changes to existing services
- ✅ `segmentio/kafka-go` is pure Go with no CGO dependency, so `CGO_ENABLED=0` Docker builds are unaffected
- ✅ KRaft removes the ZooKeeper container, keeping the Compose file manageable
- ⚠️ Kafka adds startup time (~30s); services that depend on it use `condition: service_healthy` in docker-compose so they wait for the broker to be ready
- ⚠️ A single-node broker with replication factor 1 has no fault tolerance; this is acceptable for development but must be addressed before production
- ⚠️ `segmentio/kafka-go` go.mod entries are resolved at CI time via `go get` + `go mod tidy`, consistent with the existing pattern for OTel dependencies

---

## ADR-012: Documentation-first approach

**Date:** 2026-03-18
**Status:** Accepted

**Context:**
Platform engineering projects often have great code but poor documentation. A system without documented decisions forces future contributors to reverse-engineer the "why" behind every architectural choice. Runbooks that don't exist mean on-call engineers make it up at 3am. Onboarding guides that aren't written mean every new contributor asks the same questions.

**Decision:**
Treat documentation as a first-class deliverable. Every tool has an ADR. Every alert has a runbook. The onboarding guide is written in EN + FR to make the project accessible to a broader audience. Documentation is written in parallel with code, not as an afterthought.

**Consequences:**
- ✅ ADRs create an auditable paper trail of trade-off reasoning - invaluable when revisiting decisions months later
- ✅ Runbooks encode operational knowledge: what to do when an alert fires, without needing to wake someone up
- ✅ Bilingual guides (EN + FR) lower the barrier to entry for contributors
- ✅ A well-documented repo is easier to contribute to, fork, and extend
- ⚠️ Documentation requires maintenance: ADRs can go stale if not updated when decisions change
- ⚠️ Writing good docs takes real time - it should be planned as part of every feature, not squeezed in at the end

---

## ADR-014: RabbitMQ for per-user alert routing

**Date:** 2026-03-23
**Status:** Accepted

**Context:**
Kafka handles the high-volume event pipeline efficiently, but it is a poor fit for per-user notification delivery. Kafka topics are append-only logs consumed by groups; there is no native mechanism to route a single message to a specific user without giving every user their own topic or using application-level filtering at the consumer. For user subscriptions where each user receives only the alerts matching their `{protocol, threshold}` preferences, a message broker with flexible routing is a better tool.

**Decision:**
Use RabbitMQ with a topic exchange (`onchain.alerts`, durable) for per-user alert routing.

Routing design:
- The notifier publishes `AlertMessage` payloads with routing key `user.{user_id}` whenever a HealthEvent crosses a user's subscribed threshold.
- The subscription service declares a per-user queue (`alerts.{user_id}`, auto-delete) and binds it to the exchange with binding key `user.{user_id}`.
- WebSocket connections in the subscription service consume from this queue and push JSON alerts to the browser.
- When all WebSocket connections for a user close, the auto-delete queue is removed automatically.

The Go AMQP client used is `github.com/rabbitmq/amqp091-go` (the official maintained fork of the original `streadway/amqp`).

**Why RabbitMQ alongside Kafka rather than Kafka alone:**

Kafka could theoretically route per-user messages using a topic-per-user pattern, but that does not scale: thousands of users means thousands of topics, each with its own partition metadata and log segments. Kafka is optimised for a small number of high-throughput topics, not for a large number of low-traffic per-entity queues. RabbitMQ's exchange-and-binding model is designed exactly for this routing pattern and handles queue-per-user efficiently.

**Consequences:**
- ✅ Per-user routing is expressed as a RabbitMQ binding: no application-level filtering needed in the consumer
- ✅ Auto-delete queues reclaim resources automatically when users disconnect - no manual cleanup
- ✅ Multiple WebSocket connections for the same user (multiple browser tabs) share one queue and each receives every alert
- ✅ If no WebSocket is connected for a user, the message is simply dropped - no backlog accumulates (acceptable for real-time alerts)
- ✅ `amqp091-go` is pure Go and does not affect CGO_ENABLED=0 Docker builds
- ⚠️ A single RabbitMQ node has no replication; this is acceptable for development but requires a clustered setup with quorum queues for production
- ⚠️ The notifier now has three external dependencies (Kafka, Redis, RabbitMQ); all three must be healthy before the service starts

---

## ADR-015: Redis for subscription storage

**Date:** 2026-03-23
**Status:** Accepted

**Context:**
User subscriptions (a `{user_id, protocol_id, threshold}` triple) need to be stored durably and queried efficiently by two services: the subscription service (CRUD) and the notifier (lookup by protocol at alert time). A relational database would introduce schema migrations and an ORM dependency; a document store adds operational overhead. Given the simple key-value nature of the data and the need for fast set-based lookup by protocol, Redis is a natural fit.

**Decision:**
Use Redis 7.2 as the subscription store with the following key schema:

| Key | Type | Content |
|---|---|---|
| `sub:{id}` | String | JSON-serialised `Subscription` |
| `user_subs:{user_id}` | Set | Set of subscription IDs belonging to a user |
| `proto_subs:{protocol_id}` | Set | Set of subscription IDs watching a protocol |

Creating a subscription writes to all three structures atomically via a Redis pipeline. Deleting does the same in reverse. The notifier queries `proto_subs:{protocol_id}` (a single SMEMBERS call) then fetches each matching subscription by ID to check the threshold.

Redis is configured with `maxmemory 256mb` and `allkeys-lru` eviction policy. Subscription data is small (a few hundred bytes per entry) so eviction under normal load is unlikely, but the policy ensures the container does not OOM under pathological conditions.

**Consequences:**
- ✅ SMEMBERS on `proto_subs:{protocol_id}` gives the notifier O(n) subscription lookup with a single round-trip
- ✅ Pipeline-based writes make create and delete operations atomic from the client's perspective
- ✅ Redis 7.2 Alpine is a ~30MB image with a minimal attack surface
- ✅ `github.com/redis/go-redis/v9` is the maintained community standard client for Go
- ⚠️ Subscriptions are not persisted to disk by default (no RDB or AOF configured); a Redis restart loses all subscriptions. This is acceptable for the current scope; persistence can be enabled via `redis.conf` when needed
- ⚠️ The set-based lookup means the notifier fetches each subscription individually after the SMEMBERS call (N+1 pattern). For the current scale (tens to hundreds of subscriptions per protocol) this is negligible; a Redis Hash or Lua script can optimise this if needed

