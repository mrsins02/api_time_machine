# ADR-004: Snapshot Pattern for Replay Performance

**Status:** Accepted  
**Date:** 2026-04-18

## Context

Replaying hundreds or thousands of events per request violates the < 20 ms replay target. Full replay from event 1 is O(n) in event count.

## Decision

Persist aggregate state snapshots every **N events** (default 50, configurable via `SNAPSHOT_EVERY`). A background `SnapshotBuilder` worker also fills gaps for aggregates missing up-to-date snapshots.

Replay loads the latest snapshot ≤ target, then applies only subsequent events.

## Consequences

**Positive**
- Replay cost bounded by events since last snapshot
- Predictable performance for long-lived aggregates
- Snapshots are disposable — can be rebuilt from events

**Negative**
- Additional storage per snapshot
- Snapshot/aggregate Apply logic must stay aligned
- Targeting very old versions may bypass snapshot (full replay)

## Invariants

- Snapshots never replace events as source of truth
- Deleting all snapshots must not lose data
- Snapshot version ≤ latest event version at time of creation
