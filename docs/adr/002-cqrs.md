# ADR-002: CQRS with Synchronous Projections

**Status:** Accepted  
**Date:** 2026-04-18

## Context

Current-state reads must be fast (simple SQL). Writes must maintain event-sourced history. Combining both in one model creates complexity and performance coupling.

## Decision

Implement **CQRS** with:
- **Write side:** append events + update projection in one transaction
- **Read side (current):** query materialized tables (`users`, `products`, `orders`)
- **Read side (temporal):** replay engine bypasses projections

Projections are updated **synchronously** in the write transaction, not asynchronously.

## Consequences

**Positive**
- Read-your-writes consistency
- Fast current-state queries without replay
- Clear separation of concerns

**Negative**
- Write latency includes projection cost
- Projection logic must be kept in sync with aggregate Apply logic
- Temporal reads still require replay (by design)

## Alternatives Considered

| Alternative | Rejected because |
|-------------|------------------|
| Async projections only | Stale reads after write; complex consistency guarantees |
| No projections (always replay) | Too slow for `GET /users/42` hot path |
| Event store as read model | Violates CQRS; couples reads to event schema |
