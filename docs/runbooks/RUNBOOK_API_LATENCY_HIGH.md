# Runbook: API p99 Latency High

## Alert

**Name:** API p99 Latency High  
**Severity:** warning  
**Condition:** `histogram_quantile(0.99, rate(onchain_api_request_duration_seconds_bucket[5m])) > 0.5` (p99 latency above 500ms over 5 minutes)  
**Grafana dashboard:** OnChain Health Monitor - Overview > API Latency (p99) panel

---

## What this means

The 99th percentile API response time has exceeded 500ms. This means at least 1% of requests are taking longer than half a second. This could indicate:

- The api service is CPU-throttled by its Kubernetes resource limits
- The analyzer is responding slowly (the api calls the analyzer on each request)
- A GC pause or memory pressure causing request queuing
- Abnormally large payloads being serialised

---

## Impact

- 1% of users are experiencing slow responses (>500ms)
- If left unresolved, latency may climb further and trigger cascading timeouts
- SLO budget is being consumed

---

## Investigation steps

### 1. Check current p99 latency

```bash
# In Grafana: Dashboard > OnChain Health Monitor - Overview > API Latency (p99) panel
# Or query Prometheus directly:
curl -s 'localhost:9090/api/v1/query?query=histogram_quantile(0.99,rate(onchain_api_request_duration_seconds_bucket[5m]))' | jq .
```

### 2. Check Jaeger traces for slow spans

Open [localhost:16686](http://localhost:16686):

1. Select service `onchain-api`
2. Set Min Duration to `200ms`
3. Find the slowest traces
4. Drill into the span waterfall - identify which child span is the bottleneck (api → analyzer call? JSON serialization?)

### 3. Check if the api service is CPU-throttled

```bash
# Kubernetes - check current resource usage:
kubectl top pods -n onchain-health-monitor

# Compare against limits in Helm values:
helm get values onchain-health-monitor -n onchain-health-monitor | grep -A5 resources
```

If CPU usage is at or near the limit, the pod is being throttled by the cgroup.

### 4. Check for memory pressure

```bash
# Kubernetes:
kubectl describe pod -l app=api -n onchain-health-monitor | grep -A10 "Limits\|Requests\|OOMKill"

# Docker Compose:
docker stats onchain-health-monitor-api-1
```

### 5. Check analyzer latency independently

```bash
# Time a direct call to the analyzer (bypassing the api):
time curl localhost:8082/health
```

If the analyzer itself is slow, follow the [Protocol Health Critical runbook](RUNBOOK_PROTOCOL_HEALTH_CRITICAL.md) investigation steps.

---

## Resolution

**Scale up replicas (Kubernetes):**

```bash
kubectl scale deployment/api --replicas=3 -n onchain-health-monitor
```

**Increase resource limits (Helm):**

Edit `infra/helm/services/api/values.yaml`:

```yaml
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "500m"   # increase from 200m
    memory: "256Mi"
```

Then apply:
```bash
helm upgrade onchain-health-monitor ./infra/helm/onchain-health-monitor -n onchain-health-monitor
```

**Docker Compose (local):**

Restart the api service to clear any memory/GC state:
```bash
docker-compose restart api
```

---

## Post-incident

- Document whether the latency was caused by CPU throttling, slow dependency, or payload size
- If resource limits were increased, update Helm values in the repository and merge via PR
- Review HPA (Horizontal Pod Autoscaler) thresholds - if CPU was the bottleneck, consider lowering the HPA target utilisation so scaling kicks in earlier
