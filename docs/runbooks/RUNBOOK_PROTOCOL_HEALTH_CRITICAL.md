# Runbook: Protocol Health Critical

## Alert

**Name:** Protocol Health Critical  
**Severity:** critical  
**Condition:** `onchain_analyzer_health_score < 30` for 1 minute  
**Grafana dashboard:** OnChain Health Monitor - Overview > Protocol Health Scores panel

---

## What this means

The health score of a monitored DeFi protocol has dropped below 30/100 and stayed there for at least 1 minute. This could indicate:

- Extreme price volatility on the protocol
- TVL crash (large withdrawal event)
- Protocol event anomaly in the data stream

---

## Impact

- The notifier service will be logging `ALERT` messages for the affected protocol
- API consumers will see `"status": "critical"` for this protocol

---

## Investigation steps

### 1. Identify the affected protocol

```bash
# In Grafana: Dashboard > OnChain Health Monitor - Overview > Protocol Health Scores
# Or via API:
curl localhost:8000/api/v1/protocols | jq '.protocols[] | select(.health_score < 30)'
```

### 2. Check analyzer logs

```bash
# Local (Docker Compose):
docker-compose logs -f analyzer | grep -i "critical\|score"

# Kubernetes:
kubectl logs -f deployment/analyzer -n onchain-health-monitor | grep -i "critical\|score"
```

### 3. Check collector data

```bash
# Replace <protocol_name> with the affected protocol (e.g. uniswap, aave, compound)
docker-compose logs -f collector | grep "<protocol_name>"
```

### 4. Check Jaeger traces

Open [localhost:16686](http://localhost:16686), select service `onchain-analyzer`, filter by tag `severity=critical`.

Look for spans with high latency or error tags that coincide with the alert start time.

---

## Resolution

**In mock mode (`MOCK_MODE=true`, the default):**  
Scores recover naturally within minutes as the random walk mean-reverts. No action required - wait ~2–5 minutes for the alert to auto-resolve.

**In production mode (`MOCK_MODE=false`):**  
1. Identify the RPC endpoint in use (`RPC_ENDPOINT` env var on the collector service)
2. Check if the endpoint is returning valid data: `curl $RPC_ENDPOINT/health`
3. If the endpoint is degraded, failover to a backup RPC provider
4. If the data is valid but the score is genuinely critical, escalate to the protocol risk team

---

## Post-incident

- Note time of alert and recovery in incident log
- If the alert persisted >10 minutes in mock mode, check the collector for data pipeline stalls
- If persistent in production, review RPC endpoint health and consider adding a circuit breaker
