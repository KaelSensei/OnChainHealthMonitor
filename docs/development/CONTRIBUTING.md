# Contributing - OnChain Health Monitor

This guide covers how to contribute to the OnChain Health Monitor project: branch naming, commit conventions, adding new services, adding new metrics, and code style.

---

## Branch Naming

All branches must follow this naming scheme:

```
<type>/<short-description>
```

**Types:**

| Prefix | When to use |
|--------|------------|
| `feature/` | New functionality (new endpoint, new service, new metric) |
| `fix/` | Bug fixes |
| `docs/` | Documentation only (no code change) |
| `chore/` | Maintenance tasks (dependency updates, config changes, build fixes) |
| `refactor/` | Code restructuring that doesn't change behaviour |
| `test/` | Adding or fixing tests |
| `ci/` | CI/CD pipeline changes |
| `infra/` | Terraform, Helm, or Kubernetes changes |
| `observability/` | Prometheus rules, Grafana dashboards, OTel config |

**Examples:**
```
feature/otel-instrumentation
feature/kong-rate-limiting
fix/analyzer-score-clamp
docs/technical-and-functional-documentation
chore/update-grafana-10-5
ci/github-actions-lint-workflow
infra/gke-terraform-cluster
observability/grafana-health-score-dashboard
```

**Rules:**
- Use lowercase and hyphens only (no underscores, no slashes within the description)
- Keep descriptions short (3–5 words)
- Branch from `main` unless told otherwise

---

## Conventional Commits

All commits must follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification. This enables automated changelog generation and makes the git history readable.

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | When to use |
|------|-------------|
| `feat` | A new feature |
| `fix` | A bug fix |
| `docs` | Documentation changes only |
| `chore` | Build system, dependency updates, tooling |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test` | Adding missing or correcting existing tests |
| `perf` | Performance improvements |
| `ci` | CI/CD configuration changes |
| `style` | Formatting, missing semicolons (no logic change) |
| `revert` | Reverting a previous commit |

### Scopes

Use the service or layer name as the scope:

| Scope | Applies to |
|-------|-----------|
| `collector` | `services/collector/` |
| `analyzer` | `services/analyzer/` |
| `notifier` | `services/notifier/` |
| `api` | `services/api/` |
| `prometheus` | `observability/prometheus/` |
| `grafana` | `observability/grafana/` |
| `jaeger` | `observability/jaeger/` |
| `otel` | `observability/otel/` |
| `docker` | `docker-compose.yml`, `Dockerfile`s |
| `helm` | `infra/helm/` |
| `terraform` | `infra/terraform/` |
| `k8s` | `infra/k8s/` |
| `ci` | `.github/workflows/` |

### Examples

```bash
# New feature
feat(api): add GET /api/v1/protocols/{id}/history endpoint

# Bug fix with explanation
fix(notifier): prevent duplicate alerts firing within same tick

Previously the alert loop could fire two alerts for the same protocol
if the score crossed the threshold twice during a single evaluation.
Fixes #12.

# Documentation only
docs(architecture): add data flow diagram to ARCHITECTURE.md

# Dependency or config update
chore(docker): upgrade grafana image to 10.5.0

# CI change
ci(collector): add lint and test job to GitHub Actions workflow

# Observability change
feat(prometheus): add http_request_duration_seconds histogram to api service
```

### Breaking changes

Append `!` to the type or add `BREAKING CHANGE:` in the footer:

```
feat(api)!: rename health_score field to score in protocol response

BREAKING CHANGE: consumers of GET /api/v1/protocols must update field references from health_score to score.
```

---

## How to Add a New Service

1. **Create the service directory:**
   ```bash
   mkdir -p services/myservice
   cd services/myservice
   ```

2. **Initialise the Go module:**
   ```bash
   go mod init github.com/KaelSensei/OnChainHealthMonitor/services/myservice
   ```

3. **Write `main.go`** following the existing service pattern:
   - `healthHandler` at `GET /health` → `{"status":"ok"}`
   - `metricsHandler` at `GET /metrics` → Prometheus text format
   - Goroutine for the main business loop
   - `http.Server` with `ReadTimeout` and `WriteTimeout`
   - Structured logging with `log.SetPrefix("[myservice] ")`

4. **Write the Dockerfile** (copy from an existing service and change the binary name and `EXPOSE` port):
   ```dockerfile
   FROM golang:1.22-alpine AS builder
   WORKDIR /app
   COPY go.mod ./
   COPY . .
   RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o myservice .

   FROM alpine:3.19
   RUN addgroup -S app && adduser -S app -G app
   WORKDIR /app
   COPY --from=builder /app/myservice .
   USER app
   EXPOSE <PORT>
   ENTRYPOINT ["/app/myservice"]
   ```

5. **Add the service to `docker-compose.yml`:**
   ```yaml
   myservice:
     build:
       context: ./services/myservice
       dockerfile: Dockerfile
     container_name: onchain_myservice
     ports:
       - "<PORT>:<PORT>"
     restart: unless-stopped
     healthcheck:
       test: ["CMD", "wget", "-qO-", "http://localhost:<PORT>/health"]
       interval: 10s
       timeout: 5s
       retries: 3
   ```

6. **Add a Prometheus scrape job** in `observability/prometheus/prometheus.yml`:
   ```yaml
   - job_name: "myservice"
     scrape_interval: 10s
     static_configs:
       - targets: ["myservice:<PORT>"]
     metrics_path: /metrics
     relabel_configs:
       - source_labels: [__address__]
         target_label: instance
         replacement: "myservice"
   ```

7. **Update documentation:**
   - Add the service to the services table in `README.md`
   - Add its port to the ports table in `docs/development/GETTING_STARTED.md`
   - Document its responsibility in `docs/architecture/ARCHITECTURE.md`

8. **Commit:**
   ```bash
   git add services/myservice/ docker-compose.yml observability/prometheus/prometheus.yml
   git commit -m "feat(myservice): add myservice microservice"
   ```

---

## How to Add a New Metric

Metrics are exposed in Prometheus text format from each service's `metricsHandler`. Phase 2 will switch to `prometheus/client_golang` (`promhttp`), but in Phase 1 metrics are written manually.

### Phase 1 (manual text format)

In `services/<service>/main.go`, add your metric to the `metricsHandler` function:

```go
func metricsHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "text/plain; version=0.0.4")

    // Add new metric - always include HELP and TYPE before the sample line
    out := "# HELP myservice_new_metric_total Description of the metric.\n"
    out += "# TYPE myservice_new_metric_total counter\n"
    out += fmt.Sprintf("myservice_new_metric_total %d\n", atomicValue)

    fmt.Fprint(w, out)
}
```

**Prometheus metric naming conventions:**
- Use `snake_case`
- Prefix with the service name: `collector_`, `analyzer_`, `notifier_`, `api_`
- Suffix with the unit where applicable: `_total` (counter), `_seconds` (duration), `_bytes`, `_usd`
- Counter names must end in `_total`
- Gauge names should describe what they measure (no suffix required)

### Phase 2 (promhttp)

Once `prometheus/client_golang` is added, define metrics as package-level variables:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "api_request_duration_seconds",
        Help:    "HTTP request latency by endpoint.",
        Buckets: prometheus.DefBuckets,
    }, []string{"method", "path", "status"})
)

// Replace metricsHandler with:
mux.Handle("/metrics", promhttp.Handler())
```

---

## Code Style

### Formatting

All Go code must be formatted with `gofmt` before committing:

```bash
# Format all Go files in the repo
gofmt -w ./services/...

# Or format a single service
cd services/api && gofmt -w .
```

Most editors run `gofmt` on save when the Go extension is installed.

### Dependencies

**Phase 1 rule:** No external dependencies. Each `go.mod` declares only the Go version. This keeps Docker build times fast and the dependency surface minimal.

When adding a real dependency in Phase 2+:
1. Use `go get <module>@<version>` from within the service directory
2. Justify the dependency in the PR description
3. Prefer well-maintained modules with stable APIs (e.g., `go.opentelemetry.io/otel`, `github.com/prometheus/client_golang`)

### Error handling

- Always check errors from `http.Server.ListenAndServe()` - use `log.Fatalf` for startup failures
- Use `log.Printf` for non-fatal errors inside handlers
- Never silently discard errors

### Concurrency

- Protect shared mutable state with `sync.RWMutex`
- Use `RLock`/`RUnlock` for read-only access and `Lock`/`Unlock` for writes
- Keep lock regions short - don't hold a lock while doing I/O or sleeping

### Logging

- Use `log.SetPrefix("[servicename] ")` at the top of `main()`
- Use `log.SetFlags(log.LstdFlags | log.Lmsgprefix)` for timestamp + prefix
- Log meaningful state changes: service start, score changes, alert fires
- Don't log on every metrics scrape (would flood stdout)

### HTTP handlers

- Always set `Content-Type` before writing the body
- Use `http.StatusNotFound` (404) for unknown resource IDs, not 500
- Wrap shared state reads/writes in the mutex - handlers run concurrently

---

## Pull Request Checklist

Before opening a PR:

- [ ] Branch name follows the naming convention
- [ ] All commits follow the Conventional Commits format
- [ ] Code is formatted with `gofmt`
- [ ] All services still build: `docker compose build`
- [ ] All `/health` endpoints return `{"status":"ok"}`: run the curl checks from `docs/development/GETTING_STARTED.md`
- [ ] New services or metrics are documented
- [ ] `ROADMAP.md` is updated if a milestone item was completed
