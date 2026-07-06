# Replay Engine

## Responsibility

Reconstruct aggregate state at any point in time using snapshots + event replay.

## Query Modes

| Mode | Parameter | SQL filter |
|------|-----------|------------|
| By version | `?version=8` | `version <= 8` |
| By timestamp | `?at=2026-04-18T11:00:00Z` | `created_at <= at` |
| By event ID | internal | `id <= until_id` |
| Current | no param | all events |

## Generic Replay

The engine uses a generic `replay()` function with aggregate-specific wrappers:

```go
ReplayUser(ctx, id, query)    → user.User
ReplayProduct(ctx, id, query) → product.Product
ReplayOrder(ctx, id, query)   → order.Order
```

Each aggregate implements:
- `Apply(event)` — fold event into state
- `MarshalSnapshot()` / `UnmarshalSnapshot()` — snapshot serialization

## Aggregate Apply Rules

- Events applied in strict version order
- Unknown event types ignored (forward compatibility)
- Version 0 after replay → `ErrNotFound`

## Caching Layer

`internal/cache/redis.go` wraps the replay engine:

```
cache key: replay:{type}:{id}:v:{version}
         replay:{type}:{id}:at:{nanos}
TTL: REPLAY_CACHE_TTL (default 5m)
```

Cache is bypassed when Redis is unavailable (graceful degradation).

## Metrics

`atm_replay_duration_seconds{aggregate_type}` — histogram per entity type.

## Implementation

- `internal/replay/engine.go`
- `internal/cache/redis.go`
