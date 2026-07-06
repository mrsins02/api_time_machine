# ADR-007: NATS for Event Distribution

**Status:** Accepted  
**Date:** 2026-04-18

## Context

The PRD requires a message queue for decoupled downstream processing. Services must never be directly coupled to the write path.

## Decision

Use **NATS** as the message broker. The outbox worker publishes JSON event payloads to topic `atm.events`.

Kafka was considered but NATS was chosen for:
- Simpler operational footprint in Docker Compose
- Sufficient throughput for v1 scale
- Lower latency for fan-out notifications

## Consequences

**Positive**
- Lightweight deployment (single NATS container)
- Easy to add subscribers without API changes
- Outbox pattern ensures delivery even if NATS is temporarily down

**Negative**
- Kafka ecosystem tools (Kafka Connect, etc.) not available
- May need migration to Kafka at very high scale
- Consumers must handle at-least-once delivery

## Configuration

```env
NATS_URL=nats://localhost:4222
NATS_TOPIC=atm.events
```
