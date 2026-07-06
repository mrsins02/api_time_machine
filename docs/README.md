# API Time Machine — Documentation Index

## Product

- [Product Requirements Document (PRD)](PRD.md)

## Architecture

| Document | Description |
|----------|-------------|
| [System Architecture](architecture/SYSTEM_ARCHITECTURE.md) | High-level design and package layout |
| [Database Schema](architecture/DATABASE_SCHEMA.md) | PostgreSQL tables and indexes |
| [Event Store](architecture/EVENT_STORE.md) | Append-only event log design |
| [Snapshot Strategy](architecture/SNAPSHOT_STRATEGY.md) | Checkpointing for fast replay |
| [CQRS](architecture/CQRS.md) | Command/query separation |
| [Replay Engine](architecture/REPLAY_ENGINE.md) | Temporal state reconstruction |
| [Diff Engine](architecture/DIFF_ENGINE.md) | State comparison |
| [Projection Engine](architecture/PROJECTION_ENGINE.md) | Read model maintenance |
| [REST API](architecture/REST_API.md) | HTTP endpoint reference |
| [gRPC API](architecture/GRPC_API.md) | Protobuf service reference |
| [Event Contracts](architecture/EVENT_CONTRACTS.md) | Event types and payloads |
| [Redis Strategy](architecture/REDIS_STRATEGY.md) | Replay cache design |

## Architecture Decision Records

- [ADR Index](adr/README.md)

## Quick Start

```bash
make up          # Docker: Postgres + Redis + NATS + API
open http://localhost:8080/ui/
```

Agent implementation prompts live in `docs/agents/` (gitignored).
