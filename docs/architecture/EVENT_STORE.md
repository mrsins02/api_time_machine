# Event Store Design

## Principles

1. **Append-only** — rows are never UPDATEd or DELETEd
2. **Immutable history** — the source of truth for all state
3. **Per-aggregate versioning** — monotonic version per `(aggregate_type, aggregate_id)`
4. **Optimistic concurrency** — unique constraint on version detects conflicts

## Append Flow

```go
INSERT INTO events (aggregate_type, aggregate_id, event_type, version, payload, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
```

On unique violation (`23505`) → `ErrConflict` returned to caller.

## Event Structure

```json
{
  "id": 1042,
  "aggregate_type": "user",
  "aggregate_id": "42",
  "event_type": "EmailChanged",
  "version": 3,
  "payload": { "email": "old@mail.com" },
  "metadata": {
    "actor": "admin@corp.com",
    "correlation_id": "req-abc",
    "trace_id": "trace-xyz"
  },
  "created_at": "2026-04-18T11:00:00Z"
}
```

## Query Capabilities

| Query | Implementation |
|-------|----------------|
| Events after version | `version > $after` |
| Events until version | `version <= $v` |
| Events until timestamp | `created_at <= $at` |
| Events until event ID | `id <= $until_id` |
| Search by actor | `metadata->>'actor'` |
| Search by correlation | `metadata->>'correlation_id'` |
| Timeline | ordered by version |

## Transactional Outbox

Every append also inserts into `outbox` within the same database transaction. A background worker polls unpublished rows and publishes to NATS, then marks `published = TRUE`.

This guarantees at-least-once delivery without coupling the write path to message broker availability.

## Implementation

`internal/eventstore/store.go`
