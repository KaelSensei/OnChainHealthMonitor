# Grafana Guide - OnChain Health Monitor

Grafana is pre-configured to start with dashboards and a Prometheus datasource - no manual setup required.

---

## Accessing Grafana

Once the stack is running:

```
http://localhost:3000
```

| Field    | Value   |
|----------|---------|
| Username | `admin` |
| Password | `admin` |

> **Note:** Change the admin password in production via `GF_SECURITY_ADMIN_PASSWORD`.

---

## Auto-Provisioned Dashboards

Both dashboards are automatically loaded from `observability/grafana/provisioning/dashboards/` on startup.

### OnChain Health Monitor - Overview

**UID:** `onchain-overview`  
**Purpose:** Bird's-eye view of protocol health and API performance.

| Panel | Type | What It Shows |
|-------|------|---------------|
| Protocol Health Scores | Gauge | `onchain_analyzer_health_score` per protocol (red < 30, yellow 30–70, green > 70) |
| Events Generated (rate) | Time series | `rate(onchain_collector_events_total[1m])` per protocol + event type |
| API Request Rate | Time series | `rate(onchain_api_requests_total[1m])` by path + method |
| API p99 Latency | Time series | `histogram_quantile(0.99, ...)` - SLO breach threshold at 500ms |
| Alerts Triggered (1h) | Stat | `increase(onchain_analyzer_alerts_triggered_total[1h])` per protocol |
| Notifications Sent | Stat | `onchain_notifier_notifications_sent_total` per channel |

### OnChain Health Monitor - Services

**UID:** `onchain-services-health`  
**Purpose:** Infrastructure-level view of each microservice.

| Panel | Type | What It Shows |
|-------|------|---------------|
| Service Up/Down | Stat | `up{job=~"collector|analyzer|notifier|api"}` - green = reachable |
| Last TVL by Protocol | Gauge | `onchain_collector_last_tvl_usd` - Total Value Locked in USD |
| Last Price by Protocol | Time series | `onchain_collector_last_price` - token price over time |
| Analysis Duration (p95) | Time series | `histogram_quantile(0.95, rate(onchain_analyzer_analysis_duration_seconds_bucket[5m]))` |
| Notification Latency (p95) | Time series | `histogram_quantile(0.95, rate(onchain_notifier_notification_duration_seconds_bucket[5m]))` |

---

## Adding a New Dashboard

1. Build your dashboard in the Grafana UI (localhost:3000).
2. Go to **Dashboard settings → JSON Model** and copy the JSON.
3. Save the file to `observability/grafana/provisioning/dashboards/your-dashboard.json`.
4. Set `"id": null` and assign a unique `"uid"` string.
5. Grafana will pick it up within 30 seconds (or on next restart).

> Dashboards in this folder are **read-only** in the UI - edit the JSON file directly.

---

## Alerting Rules

Alerting rules are provisioned from `observability/grafana/provisioning/alerting/rules.yaml`.

### Active Rules

| Rule | Condition | Severity | For |
|------|-----------|----------|-----|
| Protocol Health Critical | `onchain_analyzer_health_score < 30` | `critical` | 1m |
| API High Error Rate | `rate(onchain_api_requests_total{status_code=~"5.."}[5m]) > 0.01` | `warning` | 2m |
| API High p99 Latency | `histogram_quantile(0.99, ...) > 0.5s` | `warning` | 5m |

### How They Work

1. Grafana evaluates each rule query every minute against Prometheus.
2. If the condition is true for the `for` duration, the alert fires.
3. `noDataState: NoData` - no alert when there's no data (avoids false positives during startup).
4. `execErrState: Alerting` - treats query errors as firing (fail-safe).

### Adding a New Alert Rule

Add a new entry under `rules:` in `observability/grafana/provisioning/alerting/rules.yaml`:

```yaml
- uid: my-new-alert          # must be unique
  title: My New Alert
  condition: C
  data:
    - refId: A
      relativeTimeRange:
        from: 300
        to: 0
      datasourceUid: ${DS_PROMETHEUS}
      model:
        expr: your_metric > threshold
    - refId: C
      datasourceUid: __expr__
      model:
        type: classic_conditions
        conditions:
          - evaluator:
              params: [0]
              type: gt
            query:
              params: [A]
            reducer:
              type: last
            type: query
  noDataState: NoData
  execErrState: Alerting
  for: 5m
  annotations:
    summary: "What went wrong"
  labels:
    severity: warning
```

---

## Datasource

The Prometheus datasource is auto-configured:

- **Name:** `Prometheus`
- **URL:** `http://prometheus:9090` (internal Docker network)
- **UID:** `prometheus` (referenced by all dashboard panels)

No manual configuration is needed.

---

## File Structure

```
observability/grafana/
└── provisioning/
    ├── datasources/
    │   └── prometheus.yaml          # Prometheus datasource config
    ├── dashboards/
    │   ├── dashboards.yaml          # Dashboard provider config
    │   ├── onchain-overview.json       # Overview dashboard
    │   └── services-health.json     # Services dashboard
    └── alerting/
        └── rules.yaml               # SLO alerting rules
```
