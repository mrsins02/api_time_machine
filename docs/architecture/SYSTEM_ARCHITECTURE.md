# System Architecture

## High-Level Diagram

```
                    ┌─────────────┐     ┌─────────────┐
                    │  REST :8080 │     │  gRPC :9090 │
                    └──────┬──────┘     └──────┬──────┘
                           │                    │
                    ┌──────▼────────────────────▼──────┐
                    │         API Layer                │
                    │  auth · rate-limit · metrics     │
                    └──────┬───────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼────┐      ┌─────▼─────┐     ┌─────▼─────┐
    │ Command │      │  Replay   │     │   Diff    │
    │ Service │      │  Engine   │     │  Engine   │
    └────┬────┘      └─────┬─────┘     └───────────┘
         │                 │
    ┌────▼────┐      ┌─────▼─────┐     ┌───────────┐
    │ Event   │      │ Snapshot  │     │   Redis   │
    │ Store   │      │  Store    │     │   Cache   │
    └────┬────┘      └───────────┘     └───────────┘
         │
    ┌────▼────┐      ┌───────────┐
    │ Outbox  │─────▶│   NATS    │
    └────┬────┘      └───────────┘
         │
    ┌────▼────┐
    │Projection│──▶ Read Models (users, products, orders)
    └─────────┘
```

## Package Layout

```
cmd/api/              Entry point — wires all dependencies
internal/
  api/                REST handlers + embedded UI
  grpc/               gRPC TemporalService
  auth/               JWT middleware + RBAC
  cache/              Redis replay cache
  domain/             Shared types, errors, replay queries
  eventstore/         Append-only PostgreSQL event store
  snapshot/           Snapshot persistence
  replay/             Aggregate replay orchestration
  diff/               State comparison
  projection/         CQRS read-model projectors
  user|product|order/ Aggregates + command services
  platform/           Shared command append + idempotency
  worker/             Outbox publisher, snapshot builder, cache warmer
  observability/      Prometheus + OpenTelemetry
  temporal/           Query parsing helpers
pkg/
  migrate/            SQL migration runner
  events/             Cross-service event contracts
configs/              Environment configuration
migrations/           PostgreSQL schema
api/proto/            gRPC protobuf definitions
deploy/               Docker Compose + Dockerfile
docs/                 PRD, architecture, ADRs
```

## Request Flows

### Write (Command)

1. REST/gRPC handler validates request + auth
2. Command service checks expected version (optimistic lock)
3. Event appended in transaction with outbox row
4. Projector updates read model in same transaction
5. Transaction commits
6. Background worker publishes outbox → NATS
7. Snapshot worker may checkpoint aggregate asynchronously

### Read (Current)

1. Handler routes to projection read model (CQRS)
2. No event store access on hot path

### Read (Temporal)

1. Handler parses `?at=` or `?version=`
2. Replay engine loads latest snapshot ≤ target
3. Remaining events replayed in order
4. Result optionally cached in Redis
5. Response returned

## Design Principles

- **Clean Architecture** — no business logic in HTTP/gRPC handlers
- **Append-only history** — events are never updated or deleted
- **Deterministic replay** — same events always produce same state
- **Transactional consistency** — event + projection + outbox in one TX
- **Decoupled async** — NATS via outbox, never direct service calls

## Related Documents

- [Event Store](EVENT_STORE.md)
- [CQRS](CQRS.md)
- [Replay Engine](REPLAY_ENGINE.md)
- [Snapshot Strategy](SNAPSHOT_STRATEGY.md)
