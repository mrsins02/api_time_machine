# ADR-006: Redis for Replay Result Caching

**Status:** Accepted  
**Date:** 2026-04-18

## Context

Temporal queries for the same `(aggregate, version)` are often repeated — especially in the UI replay animation and diff endpoints. Replaying from PostgreSQL on every request wastes CPU.

## Decision

Cache replay results in **Redis** with configurable TTL (`REPLAY_CACHE_TTL`, default 5 minutes). Cache keys encode aggregate type, ID, and version or timestamp.

If Redis is unavailable, the system operates without cache (graceful degradation).

## Consequences

**Positive**
- Sub-millisecond cache hits for repeated temporal queries
- Redis failure does not affect correctness
- Cache warmer pre-populates hot aggregates

**Negative**
- Stale cache possible within TTL after new events (acceptable for read-heavy temporal queries)
- Additional infrastructure dependency
- Cache invalidation is TTL-based, not event-driven (v1)

## Future

Event-driven invalidation via NATS subscription when new events are published.
