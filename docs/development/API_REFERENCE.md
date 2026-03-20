# OnChain Health Monitor - API Reference

Complete reference for the OnChain Health Monitor REST API.

---

## Base URLs

| Access Method       | URL                          | When to Use                                    |
|---------------------|------------------------------|------------------------------------------------|
| Kong API Gateway    | `http://localhost:8000`      | Default - rate limiting, auth, and logging applied |
| Direct API          | `http://localhost:8080`      | Development / internal health checks only      |
| Swagger UI          | `http://localhost:8090/swagger` | Interactive API docs and request builder    |
| Kong Swagger route  | `http://localhost:8000/swagger` | Swagger UI proxied through Kong             |

> **Production note:** External traffic should always go through Kong (`8000`). Direct access (`8080`) is intentionally kept for Docker health checks and internal service communication.

---

## Endpoints

| Method | Path                        | Tag       | Description                             |
|--------|-----------------------------|-----------|-----------------------------------------|
| GET    | `/health`                   | health    | Service liveness check                  |
| GET    | `/api/v1/protocols`         | protocols | List all monitored DeFi protocols       |
| GET    | `/api/v1/protocols/{id}`    | protocols | Get a specific protocol by slug         |

---

## Endpoint Details & Examples

### `GET /health`

Returns service status. Used by Docker health checks and load balancers.

**Request:**
```bash
curl http://localhost:8000/health
# or direct:
curl http://localhost:8080/health
```

**Response `200 OK`:**
```json
{
  "status": "ok"
}
```

---

### `GET /api/v1/protocols`

Returns health scores and metadata for all monitored DeFi protocols.

**Request:**
```bash
curl http://localhost:8000/api/v1/protocols
```

**Response `200 OK`:**
```json
{
  "protocols": [
    {
      "id": "uniswap",
      "name": "Uniswap",
      "health_score": 87,
      "status": "healthy",
      "tvl_usd": 4523891234.56,
      "price_usd": 12.34,
      "last_updated": "2026-03-18T21:00:00Z"
    },
    {
      "id": "aave",
      "name": "Aave",
      "health_score": 72,
      "status": "healthy",
      "tvl_usd": 9812345678.90,
      "price_usd": 98.76,
      "last_updated": "2026-03-18T21:00:00Z"
    },
    {
      "id": "compound",
      "name": "Compound",
      "health_score": 25,
      "status": "critical",
      "tvl_usd": 1234567890.12,
      "price_usd": 45.67,
      "last_updated": "2026-03-18T21:00:00Z"
    }
  ],
  "count": 3
}
```

---

### `GET /api/v1/protocols/{id}`

Returns health score and metadata for a specific DeFi protocol by its slug.

**Parameters:**

| Name | In   | Required | Type   | Description              |
|------|------|----------|--------|--------------------------|
| `id` | path | yes      | string | Protocol slug (e.g. `uniswap`) |

**Request:**
```bash
curl http://localhost:8000/api/v1/protocols/uniswap
curl http://localhost:8000/api/v1/protocols/aave
curl http://localhost:8000/api/v1/protocols/compound
```

**Response `200 OK`:**
```json
{
  "id": "uniswap",
  "name": "Uniswap",
  "health_score": 87,
  "status": "healthy",
  "tvl_usd": 4523891234.56,
  "price_usd": 12.34,
  "last_updated": "2026-03-18T21:00:00Z"
}
```

**Response `404 Not Found`:**
```json
{
  "error": "protocol not found"
}
```

---

## Data Models

### Protocol

| Field          | Type    | Description                                              |
|----------------|---------|----------------------------------------------------------|
| `id`           | string  | Unique slug identifier (e.g. `uniswap`)                 |
| `name`         | string  | Human-readable protocol name                            |
| `health_score` | integer | Health score 0–100 (0 = critical, 100 = fully healthy)  |
| `status`       | string  | Derived label: `healthy`, `degraded`, or `critical`     |
| `tvl_usd`      | float   | Total Value Locked in USD                               |
| `price_usd`    | float   | Protocol token price in USD                             |
| `last_updated` | string  | ISO 8601 timestamp of last data update                  |

**Health score thresholds:**

| Score Range | Status     |
|-------------|------------|
| 70 – 100    | `healthy`  |
| 30 – 69     | `degraded` |
| 0 – 29      | `critical` |

### ProtocolListResponse

| Field       | Type             | Description                      |
|-------------|------------------|----------------------------------|
| `protocols` | array of Protocol | All currently monitored protocols |
| `count`     | integer          | Total number of protocols         |

---

## Error Format

All errors follow a consistent JSON shape:

```json
{
  "error": "human-readable error message"
}
```

| HTTP Status | Meaning               | Example                         |
|-------------|-----------------------|---------------------------------|
| `404`       | Resource not found    | `{"error":"protocol not found"}` |
| `500`       | Internal server error | `{"error":"internal server error"}` |

---

## Rate Limiting

All requests through Kong are subject to rate limiting. The following headers are included in every response:

| Header                          | Description                              |
|---------------------------------|------------------------------------------|
| `X-RateLimit-Limit-Minute`      | Maximum requests allowed per minute      |
| `X-RateLimit-Remaining-Minute`  | Remaining requests in the current minute |

Default limits: **60 requests/minute**, **1000 requests/hour**.

When a limit is exceeded, Kong returns `429 Too Many Requests`.

Other Kong-injected headers:

| Header          | Description                            |
|-----------------|----------------------------------------|
| `X-Request-ID`  | Unique request correlation ID (UUID)   |
| `X-Powered-By`  | `OnChain-Health-Monitor`                  |
| `X-Gateway`     | `Kong`                                 |

---

## OpenAPI Spec & Swagger UI

The API is fully described in an OpenAPI 3.0.3 specification.

**Spec location:** `infra/api-spec/openapi.yaml`

**Swagger UI:** `http://localhost:8090/swagger`  
_(also accessible via Kong at `http://localhost:8000/swagger`)_

### Updating the spec

1. Edit `infra/api-spec/openapi.yaml`
2. Restart the Swagger UI container to pick up changes:
   ```bash
   docker-compose restart swagger-ui
   ```
3. Reload `http://localhost:8090/swagger` in your browser

The spec file is mounted as a read-only volume into the Swagger UI container - no rebuild needed, just a container restart.

---

## Running the API locally

```bash
# Start full stack
docker-compose up --build

# Verify API
curl http://localhost:8000/api/v1/protocols

# Open interactive docs
open http://localhost:8090/swagger
```
