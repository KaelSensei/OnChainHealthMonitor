# Runbook: API High Error Rate

## Alert

**Name:** API High Error Rate  
**Severity:** warning  
**Condition:** `rate(onchain_api_requests_total{status_code=~"5.."}[5m]) > 0.01` (more than 1 5xx per 100 requests over 5 minutes)  
**Grafana dashboard:** OnChain Health Monitor - Overview > API Error Rate panel

---

## What this means

The API service is returning HTTP 5xx errors at a rate above the SLO threshold of 1%. This could indicate:

- A panic or unhandled error in the api service
- The analyzer service being unreachable (the api reads scores from the analyzer)
- Resource exhaustion (OOM, CPU throttle)
- A dependency (Redis, database) being unavailable - not applicable in current mock setup

---

## Impact

- External API consumers will see `500 Internal Server Error` or `503 Service Unavailable` responses
- Health score data may be stale or unavailable
- Kong access logs will record elevated 5xx counts

---

## Investigation steps

### 1. Check the current error rate

```bash
# In Grafana: Dashboard > OnChain Health Monitor - Overview > API Error Rate
# Or query Prometheus directly:
curl -s 'localhost:9090/api/v1/query?query=rate(onchain_api_requests_total{status_code=~"5.."}[5m])' | jq .
```

### 2. Check api service logs for panics or errors

```bash
# Local (Docker Compose):
docker-compose logs -f api | grep -i "error\|panic\|fatal"

# Kubernetes:
kubectl logs -f deployment/api -n onchain-health-monitor | grep -i "error\|panic\|fatal"
```

### 3. Check if upstream services are healthy

```bash
# Analyzer (the api depends on analyzer for health scores):
curl localhost:8082/health

# Collector:
curl localhost:8081/health
```

### 4. Check Kong access logs

```bash
# Kong admin API - check recent request logs:
curl localhost:8001/

# Kong access logs via Docker:
docker-compose logs -f kong | grep " 5[0-9][0-9] "
```

### 5. Check Jaeger traces for failed requests

Open [localhost:16686](http://localhost:16686), select service `onchain-api`, filter by tag `error=true`.

---

## Resolution

**Restart the api service:**

```bash
# Docker Compose:
docker-compose restart api

# Kubernetes:
kubectl rollout restart deployment/api -n onchain-health-monitor
```

**If panics are in the logs:**
1. Note the panic message and stack trace
2. Identify the handler function that panicked
3. Check for nil pointer dereferences in the analyzer response parsing code
4. Add a recovery middleware if not already present

**If the analyzer is unreachable:**
1. Restart the analyzer: `docker-compose restart analyzer`
2. Verify connectivity: `docker-compose exec api wget -qO- http://analyzer:8082/health`

---

## Post-incident

- Document the root cause and resolution in the incident log
- If a panic was found, open a bug ticket with the stack trace
- Review whether the error budget has been consumed and whether a freeze is warranted
