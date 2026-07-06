# API Time Machine — Product Requirements Document

## Overview

API Time Machine is a production-grade **Temporal REST API Platform** that replays any entity to any point in time. Clients retrieve the exact historical state of a resource using a timestamp or version.

**Example**

```
GET /users/42?at=2026-04-18T13:22:51Z
```

Returns the user as they existed at that moment — even if the email has since changed.

## Vision

This is **not** an audit log, soft delete, or simple versioning layer. It is a **Temporal Database implemented at the application layer**:

- Every change is immutable
- Every state is reproducible from events
- Every request can travel through time

## Goals

| Capability | Description |
|------------|-------------|
| Event Sourcing | Append-only immutable event log per aggregate |
| CQRS | Writes go to event store; reads use projections |
| Snapshotting | Periodic state checkpoints for fast replay |
| Event Replay | Reconstruct state at version, timestamp, or event |
| Temporal Queries | `?at=` and `?version=` on all entity endpoints |
| Diff Engine | Compare two points in time field-by-field |
| Optimistic Locking | `If-Match` / expected version on writes |
| High-Performance Replay | Target < 20 ms via snapshots + Redis cache |

## Entities (v1)

| Aggregate | ID prefix | Key events |
|-----------|-----------|------------|
| User | `users` | Created, Updated, Deleted, EmailChanged, … |
| Product | `products` | Created, Updated, Deleted, PriceChanged |
| Order | `orders` | Created, Updated, Cancelled, ItemAdded, ItemRemoved |

## API Surface

### REST (`:8080`)

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/{entity}/{id}` | Current state |
| GET | `/{entity}/{id}?at=` | State at timestamp |
| GET | `/{entity}/{id}?version=` | State at version |
| GET | `/{entity}/{id}/history` | Full event log |
| GET | `/{entity}/{id}/timeline` | Version timeline |
| GET | `/{entity}/{id}/diff?from=&to=` | Field diff |
| POST | `/{entity}/{id}/replay` | Explicit replay |
| POST | `/{entity}/{id}/preview` | Non-destructive rollback preview |
| GET | `/events` | Search events |
| POST | `/auth/token` | JWT issuance (when auth enabled) |
| GET | `/ui/` | Interactive dark UI |

### gRPC (`:9090`)

`TemporalService` — Get/Replay for users, products, orders; timeline; event search.

## Non-Goals (v1)

- Destructive rollback (history is never rewritten)
- Branch timelines / alternate histories
- Temporal SQL (`AS OF` syntax)

## Success Criteria

1. Historical state matches replay from events (deterministic)
2. Concurrent writers never corrupt history (optimistic locking)
3. Replay p99 < 20 ms for aggregates with snapshots
4. All writes produce outbox entries for async consumers
5. UI allows animated scrub through entity versions

## Tech Stack

| Layer | Choice |
|-------|--------|
| Language | Go 1.23 |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Queue | NATS |
| RPC | gRPC + protobuf |
| API | REST (chi) |
| Auth | JWT + RBAC |
| Observability | Prometheus + OpenTelemetry |

## Related Documents

- [System Architecture](architecture/SYSTEM_ARCHITECTURE.md)
- [ADRs](adr/README.md)
- [Database Schema](architecture/DATABASE_SCHEMA.md)
