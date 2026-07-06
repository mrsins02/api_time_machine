# Redis Strategy

## Use Cases

| Use Case | Implementation | Status |
|----------|----------------|--------|
| Replay result cache | `internal/cache/redis.go` | Active |
| Idempotency | PostgreSQL (not Redis) | By design |
| Session store | N/A | Not needed (JWT) |

## Replay Cache

### Key Format

```
replay:{aggregate_type}:{aggregate_id}:v:{version}
replay:{aggregate_type}:{aggregate_id}:at:{unix_nanos}
```

### TTL

`REPLAY_CACHE_TTL` environment variable (default `5m`).

### Flow

```
1. Check Redis for key
2. Hit → deserialize JSON, return
3. Miss → replay from events, SET with TTL, return
```

### Graceful Degradation

If Redis is unavailable at startup, the cache layer is disabled and all replays hit PostgreSQL directly. No write failures occur.

## Cache Warmer

`internal/worker/cache_warmer.go` periodically replays frequently-accessed aggregates to pre-populate cache. Runs every 60 seconds, processes up to 50 aggregate IDs.

## Configuration

```env
REDIS_URL=redis://localhost:6379/0
REPLAY_CACHE_TTL=5m
```

## Future Considerations

- Cache invalidation on new events (pub/sub from outbox)
- Compression for large aggregate states
- Redis Cluster for multi-node deployments

## ADR

See [ADR-006: Redis Replay Cache](../adr/006-redis-replay-cache.md)
