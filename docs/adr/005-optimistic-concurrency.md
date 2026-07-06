# ADR-005: Optimistic Concurrency Control

**Status:** Accepted  
**Date:** 2026-04-18

## Context

Multiple writers may update the same aggregate concurrently. Global locks do not scale and create availability bottlenecks.

## Decision

Use **optimistic locking** via per-aggregate version numbers:
- Each event has monotonically increasing `version`
- Unique constraint on `(aggregate_type, aggregate_id, version)`
- Clients send expected version via `If-Match` header or `expected_version` query param
- Mismatch returns `412 Precondition Failed`

Additionally, `Idempotency-Key` header deduplicates retried commands.

## Consequences

**Positive**
- No global locks; high write throughput
- Conflict detection is precise and debuggable
- Idempotency prevents duplicate side effects on retry

**Negative**
- Clients must handle 412 and retry with fresh version
- Hot aggregates may see higher conflict rates

## Implementation

- Version check: `internal/platform/commands.go`
- Idempotency: `idempotency_keys` table
