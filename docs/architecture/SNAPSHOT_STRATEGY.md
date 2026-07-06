# Snapshot Strategy

## Purpose

Replaying hundreds of events per request is too slow. Snapshots store the aggregate state at a specific version so replay only needs events **after** the snapshot.

## When Snapshots Are Created

| Trigger | Mechanism |
|---------|-----------|
| Every N events | `SNAPSHOT_EVERY` env (default 50) on write path |
| Background worker | `SnapshotBuilder` polls aggregates missing up-to-date snapshots every 30s |

## Replay Algorithm

```
1. Load latest snapshot for aggregate (if any)
2. If target version < snapshot version → full replay from v0
3. Else start from snapshot.state at snapshot.version
4. Load events where version > snapshot.version AND ≤ target
5. Apply each event in order
6. Return final state
```

## Storage

```sql
INSERT INTO snapshots (aggregate_type, aggregate_id, version, state)
VALUES ($1, $2, $3, $4)
ON CONFLICT (aggregate_type, aggregate_id, version) DO UPDATE SET state = EXCLUDED.state
```

State is JSON-serialized aggregate (same shape as API response).

## Performance Target

| Scenario | Target |
|----------|--------|
| With snapshot, < 50 events after | < 5 ms |
| Full replay, 100 events | < 20 ms |
| Cold replay, 1000+ events | snapshot worker should have checkpointed |

## Future: Compression

Snapshots may be compressed (gzip) in `state` column. Event compression (squashing repetitive events) is a planned enhancement — snapshots make this transparent to readers.

## Implementation

- `internal/snapshot/store.go`
- `internal/worker/outbox.go` (SnapshotBuilder)
