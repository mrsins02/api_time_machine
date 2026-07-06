# ADR-003: Transactional Outbox for Async Messaging

**Status:** Accepted  
**Date:** 2026-04-18

## Context

Downstream services (analytics, search indexing, notifications) must react to new events. Direct HTTP calls from the write path create tight coupling and failure modes where events are stored but not delivered.

## Decision

Use the **Transactional Outbox** pattern:
1. Insert event + outbox row in the same DB transaction
2. Background worker polls `outbox WHERE published = FALSE`
3. Worker publishes to NATS and marks row published

## Consequences

**Positive**
- At-least-once delivery guaranteed relative to event persistence
- Write path independent of NATS availability
- Easy to add new consumers without changing write path

**Negative**
- Delivery latency (poll interval ~2s)
- Duplicate messages possible (consumers must be idempotent)
- Outbox table requires monitoring for lag

## Implementation

- Table: `outbox`
- Worker: `internal/worker/outbox.go`
- Topic: `atm.events` (configurable via `NATS_TOPIC`)
