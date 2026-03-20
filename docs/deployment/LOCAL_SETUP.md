# Local Setup - OnChain Health Monitor

This guide covers the full local deployment using Docker Compose, including troubleshooting common issues and reset procedures.

---

## Prerequisites

- Docker Engine 24.x or later
- Docker Compose v2 (`docker compose`) - bundled with Docker Desktop on macOS/Windows; install via `apt install docker-compose-plugin` on Linux
- 4 GB of free RAM recommended (Grafana and Prometheus are memory-hungry)
- Ports `3000`, `4317`, `4318`, `8080–8083`, `9090`, `16686` must be free on your host

---

## Full Stack Start

```bash
# Clone the repo (skip if already cloned)
git clone https://github.com/KaelSensei/OnChainHealthMonitor.git
cd OnChainHealthMonitor

# Build all images and start all 7 containers
docker compose up --build

# Or run in the background (detached)
docker compose up --build -d
```

On first run, Docker will:
1. Pull `golang:1.22-alpine`, `alpine:3.19`, `prom/prometheus:v2.51.0`, `grafana/grafana:10.4.0`, `jaegertracing/all-in-one:1.56`
2. Compile all 4 Go services via multi-stage builds
3. Start all containers in dependency order

**Expected startup time:** ~2 minutes on first run, ~10 seconds on subsequent runs.

---

## Verify the Stack

Once all containers are running:

```bash
# Application services
curl http://localhost:8080/health   # API        → {"status":"ok"}
curl http://localhost:8081/health   # Collector  → {"status":"ok"}
curl http://localhost:8082/health   # Analyzer   → {"status":"ok"}
curl http://localhost:8083/health   # Notifier   → {"status":"ok"}

# Observability UIs (open in browser)
# Prometheus:  http://localhost:9090
# Grafana:     http://localhost:3000  (admin / admin)
# Jaeger:      http://localhost:16686

# Confirm Prometheus is scraping the services
curl 'http://localhost:9090/api/v1/targets' | jq '.data.activeTargets[].labels'
```

---

## Environment Variable Overrides

Docker Compose environment variables can be overridden using a `.env` file in the project root, or by passing `--env-file` to `docker compose`.

Create a `.env` file:

```bash
# .env (not committed - add to .gitignore)
GF_SECURITY_ADMIN_PASSWORD=mysecretpassword
```

To override Grafana's admin password:
```bash
GF_SECURITY_ADMIN_PASSWORD=mysecretpassword docker compose up -d
```

To override the alert threshold for the notifier (once env var support is added in Phase 2):
```bash
ALERT_THRESHOLD=50 docker compose up -d notifier
```

To run only specific services (e.g., just the app without observability):
```bash
docker compose up collector analyzer notifier api
```

---

## Stopping and Resetting

### Stop (keep data)

```bash
docker compose down
```

Containers are stopped and removed, but the `grafana_data` volume is preserved. On next `docker compose up`, Grafana will remember dashboards and users.

### Full reset (delete all data)

```bash
# Remove containers AND volumes (wipes Grafana state, Prometheus TSDB)
docker compose down -v
```

### Rebuild a single service

```bash
# Rebuild only the collector image and restart its container
docker compose up --build --no-deps collector
```

### Force a clean rebuild of all images

```bash
docker compose build --no-cache
docker compose up
```

---

## Troubleshooting

### Port already in use

**Symptom:**
```
Error starting userland proxy: listen tcp 0.0.0.0:8080: bind: address already in use
```

**Fix:** Find and kill the process using the port:
```bash
# macOS / Linux
lsof -i :8080
kill -9 <PID>

# Or stop Docker and find the conflicting service
```

If you can't free the port, override it in `docker-compose.yml`:
```yaml
ports:
  - "9080:8080"   # host:container
```

### Docker out of memory

**Symptom:** Containers crash with `exit code 137` or Grafana/Prometheus fail to start.

**Fix (Docker Desktop):**
1. Open Docker Desktop → Settings → Resources
2. Increase Memory to at least 4 GB
3. Restart Docker Desktop

**Fix (Linux):** Check available memory:
```bash
free -h
docker stats   # Monitor container memory usage
```

### Build fails: `go: command not found` or similar

This happens inside the Docker build. Ensure the `golang:1.22-alpine` image is pulled correctly:
```bash
docker pull golang:1.22-alpine
docker compose build --no-cache collector
```

### Service fails to start: `depends_on` timing

**Symptom:** `analyzer` starts before `collector`'s HTTP server is ready, logs connection refused.

**Fix:** Add a health check wait in the service or increase the `depends_on` condition to `service_healthy` (requires health check to be defined, which is already the case in `docker-compose.yml`). If issues persist:
```bash
# Start observability first, then app services
docker compose up -d prometheus grafana jaeger
docker compose up collector
docker compose up analyzer notifier api
```

### Prometheus shows "0 targets" or "unhealthy"

**Check:**
1. Confirm all 4 app containers are running: `docker compose ps`
2. Check Prometheus config is mounted: `docker compose exec prometheus cat /etc/prometheus/prometheus.yml`
3. Verify DNS resolution: `docker compose exec prometheus wget -O- http://collector:8081/metrics`

If DNS resolution fails, all containers must be on the same network. Ensure you're using `docker compose` (v2), not running containers independently.

### Grafana shows "Datasource not found"

Prometheus is automatically available as a data source only if provisioned. For Phase 1:
1. Go to **Grafana → Connections → Data sources → Add data source**
2. Select **Prometheus**
3. URL: `http://prometheus:9090`
4. Click **Save & Test**

### Container logs show no data / service crashed

```bash
# Check container status
docker compose ps

# View logs for a specific service
docker compose logs --tail=50 analyzer

# Restart a single service
docker compose restart notifier
```

### Disk space issues

Repeated `--no-cache` builds accumulate dangling images. Clean up:
```bash
# Remove dangling images
docker image prune

# Remove all unused images (more aggressive)
docker image prune -a

# Full system prune (removes stopped containers, networks, dangling images)
docker system prune
```

---

## Running Without Docker (native Go)

Each service runs standalone:
```bash
cd services/collector && go run .   # :8081
cd services/analyzer  && go run .   # :8082
cd services/notifier  && go run .   # :8083
cd services/api       && go run .   # :8080
```

Run Prometheus and Grafana with Docker while pointing at `host.docker.internal`:
```yaml
# In prometheus.yml, replace collector:8081 with:
targets: ["host.docker.internal:8081"]
```

---

## Useful Commands Reference

```bash
# Start full stack
docker compose up --build

# Start detached
docker compose up -d

# View all container status
docker compose ps

# Follow all logs
docker compose logs -f

# Follow specific service logs
docker compose logs -f api

# Restart a service
docker compose restart analyzer

# Stop all
docker compose down

# Full reset including volumes
docker compose down -v

# Rebuild one service without restarting others
docker compose up --build --no-deps collector

# Execute a command inside a container
docker compose exec api sh

# Check Prometheus targets via API
curl http://localhost:9090/api/v1/targets | jq .
```
