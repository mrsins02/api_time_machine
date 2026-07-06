# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for API Time Machine.

## Format

Each ADR follows the pattern: **Context → Decision → Consequences**

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [001](001-event-sourcing.md) | Event Sourcing as Source of Truth | Accepted |
| [002](002-cqrs.md) | CQRS with Synchronous Projections | Accepted |
| [003](003-transactional-outbox.md) | Transactional Outbox for Async Messaging | Accepted |
| [004](004-snapshot-pattern.md) | Snapshot Pattern for Replay Performance | Accepted |
| [005](005-optimistic-concurrency.md) | Optimistic Concurrency Control | Accepted |
| [006](006-redis-replay-cache.md) | Redis for Replay Result Caching | Accepted |
| [007](007-nats-messaging.md) | NATS for Event Distribution | Accepted |
| [008](008-jwt-rbac.md) | JWT + RBAC for API Authorization | Accepted |

## Related

- [PRD](../PRD.md)
- [System Architecture](../architecture/SYSTEM_ARCHITECTURE.md)
