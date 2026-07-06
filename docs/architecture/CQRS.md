# CQRS Design

## Separation

| Side | Responsibility | Storage |
|------|----------------|---------|
| **Command** | Validate, append events, update projections | `events` + read models |
| **Query (current)** | Read materialized view | `users`, `products`, `orders` |
| **Query (temporal)** | Replay to point in time | `events` + `snapshots` |

## Write Path

```
Command → Event Store (append) → Projector (same TX) → Read Model
```

All three happen in a **single PostgreSQL transaction**. If projection fails, the event is not committed.

## Read Path

### Current State
```
GET /users/42  →  SELECT * FROM users WHERE id = '42'
```
Never touches the event store.

### Historical State
```
GET /users/42?version=3  →  Replay Engine  →  events + snapshots
```
Bypasses read model — state is derived from events.

## Projectors

Each aggregate has a projector implementing:

```go
Apply(ctx, tx, event) error      // upsert read model in transaction
GetCurrent(ctx, id) (*Entity, error)  // query read model
```

| Projector | Table | Package |
|-----------|-------|---------|
| UserProjector | users | `internal/projection/user.go` |
| ProductProjector | products | `internal/projection/catalog.go` |
| OrderProjector | orders | `internal/projection/catalog.go` |

## Consistency Model

- **Read-your-writes**: guaranteed within same request (transactional projection)
- **Eventually consistent async**: NATS consumers may lag; outbox lag exposed via `atm_projection_lag_events` metric
- **Temporal reads**: always consistent — derived from immutable event log

## Why Not Project Historical State?

Temporal queries replay events rather than maintaining per-version read models. This avoids combinatorial storage growth while snapshots keep replay fast.
