# ADR-001: Event Sourcing as Source of Truth

**Status:** Accepted  
**Date:** 2026-04-18

## Context

The platform must reconstruct any entity at any historical point with guaranteed correctness. Traditional CRUD with audit tables cannot provide deterministic replay or version-addressable state.

## Decision

All aggregate state changes are stored as **immutable, append-only events** in PostgreSQL. Current state is always derivable by replaying events (optionally from a snapshot). The `events` table is never UPDATEd or DELETEd.

## Consequences

**Positive**
- Complete audit trail by construction
- Deterministic temporal queries
- Natural support for replay, diff, and timeline APIs

**Negative**
- Read path for historical queries is more expensive than a simple SELECT
- Event schema evolution requires careful forward-compatibility
- Storage grows monotonically (mitigated by snapshots + future archival)

## Alternatives Considered

| Alternative | Rejected because |
|-------------|------------------|
| Audit log + CRUD | Cannot guarantee replay correctness; dual writes |
| Temporal tables (SQL) | DB-vendor-specific; less control over event semantics |
| Versioned rows | Conflates current and historical; poor event semantics |
