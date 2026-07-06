# Projection Engine

## Overview

Projectors maintain CQRS read models by folding events into SQL tables. They run **synchronously** inside the write transaction.

## Lifecycle

```
Event appended → Projector.Apply(event) → UPSERT read model
```

## User Projection Example

```sql
INSERT INTO users (id, name, email, version, deleted, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  email = EXCLUDED.email,
  version = EXCLUDED.version,
  deleted = EXCLUDED.deleted,
  updated_at = EXCLUDED.updated_at
```

The projector first calls `aggregate.Apply(event)` in memory to derive the new state, then persists it.

## Idempotency

Re-applying the same event version produces the same read model row (UPSERT semantics). This supports safe replay tooling.

## Lag Monitoring

Unpublished outbox rows indicate async pipeline lag:

```
atm_projection_lag_events  — gauge of unpublished outbox entries
```

## Package Structure

| File | Projector |
|------|-----------|
| `internal/projection/user.go` | users table |
| `internal/projection/catalog.go` | products + orders tables |

## Interface

Command services depend on a `Projector` interface (defined per aggregate package) to avoid import cycles:

```go
type Projector interface {
    Apply(ctx, tx, event) error
    GetCurrent(ctx, id) (*Entity, error)
}
```
