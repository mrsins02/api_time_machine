# REST API

Base URL: `http://localhost:8080`

## Authentication

When `JWT_SECRET` is set:

```bash
# Issue token
POST /auth/token
{ "subject": "dev", "role": "admin" }

# Use token
Authorization: Bearer <token>
```

Roles: `admin`, `writer`, `reader`

| Role | Permissions |
|------|-------------|
| admin | all |
| writer | create, update, delete |
| reader | read, replay, diff, history |

Public (no auth): `/health`, `/metrics`, `/ui/*`

## Headers

| Header | Purpose |
|--------|---------|
| `Authorization` | Bearer JWT |
| `If-Match` | Expected version for optimistic locking |
| `Idempotency-Key` | Deduplicate command retries |
| `X-Actor` | Stored in event metadata |
| `X-Correlation-ID` | Request tracing |
| `X-Trace-ID` | Distributed trace ID |

## Users

```
POST   /users              Create
GET    /users/{id}         Read (current or ?at= / ?version=)
PUT    /users/{id}         Update
DELETE /users/{id}         Soft delete (UserDeleted event)
GET    /users/{id}/history
GET    /users/{id}/timeline
GET    /users/{id}/diff?from=&to=
POST   /users/{id}/replay  { "at": "...", "version": N }
POST   /users/{id}/preview Non-destructive rollback preview
```

## Products

```
POST   /products
GET    /products/{id}
PUT    /products/{id}
DELETE /products/{id}
GET    /products/{id}/history|timeline|diff
POST   /products/{id}/replay
```

## Orders

```
POST   /orders
PUT    /orders/{id}
POST   /orders/{id}/cancel
POST   /orders/{id}/items    { "item": { "product_id", "quantity", "price_cents" } }
GET    /orders/{id}
GET    /orders/{id}/history|timeline|diff
POST   /orders/{id}/replay
```

## Event Search

```
GET /events?aggregate_type=user&aggregate_id=42&event_type=EmailChanged&actor=admin&from=&to=
```

## Error Responses

```json
{ "error": "expected version mismatch" }
```

| Status | Meaning |
|--------|---------|
| 409 | Version conflict / duplicate create |
| 412 | Expected version mismatch |
| 404 | Aggregate not found |
| 401 | Missing/invalid JWT |
| 403 | Insufficient role |
| 429 | Rate limit exceeded |

## Rate Limiting

300 requests per minute per IP.

## Implementation

`internal/api/`
