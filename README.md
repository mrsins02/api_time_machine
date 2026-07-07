# API Time Machine

A **temporal REST API platform** that stores every change as an immutable event and can reconstruct any entity at any point in time — by timestamp or version.

```http
GET /users/42?at=2026-04-18T13:22:51Z
```

Returns user `42` exactly as they existed at that moment, even if the email has since changed.

## What it does

| Capability | Description |
|------------|-------------|
| Event sourcing | Append-only event log per aggregate (user, product, order) |
| Temporal queries | `?at=` and `?version=` on every entity endpoint |
| Replay & diff | Reconstruct state or compare two points in time field-by-field |
| CQRS | Writes go to the event store; reads use projections |
| Snapshots + Redis | Fast replay for aggregates with many events |

This is **not** an audit log or soft-delete layer. History is never rewritten — every state is reproducible from events.

## Quick start

**Requirements:** Docker, or Go 1.25+ with local Postgres, Redis, and NATS.

```bash
make up          # start Postgres, Redis, NATS, and the API
open http://localhost:8080/ui/
```

The API listens on `:8080` (REST + UI) and `:9090` (gRPC).

Stop the stack:

```bash
make down
```

## Tutorial: change `users/42` and see the history

This walkthrough creates a user, updates them several times, then travels back through every version.

### 1. Get a token (Docker dev stack)

When `JWT_SECRET` is set (it is in `deploy/docker-compose.yml`), write endpoints require a Bearer token:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/auth/token \
  -H 'Content-Type: application/json' \
  -d '{"subject":"dev","role":"admin"}' | jq -r .token)
```

### 2. Create user `42`

```bash
curl -s -X POST http://localhost:8080/users \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"id":"42","name":"Ada Lovelace","email":"ada@example.com"}' | jq
```

Each write appends an event and bumps `version`. Note the `version` in the response (starts at `1`).

### 3. Change the user

```bash
curl -s -X PUT http://localhost:8080/users/42 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada.lovelace@example.com"}' | jq
```

Make more changes if you like — name, address, avatar. Every `PUT` creates a new immutable event; nothing is overwritten.

### 4. See the current state

```bash
curl -s http://localhost:8080/users/42 | jq
```

### 5. Travel back in time

**By version** — replay user `42` as they were at version 1 (right after creation):

```bash
curl -s 'http://localhost:8080/users/42?version=1' | jq
```

**By timestamp** — copy `updated_at` from an earlier response and query that instant:

```bash
curl -s 'http://localhost:8080/users/42?at=2026-04-18T13:22:51Z' | jq
```

**Explicit replay** (POST body):

```bash
curl -s -X POST http://localhost:8080/users/42/replay \
  -H 'Content-Type: application/json' \
  -d '{"version": 1}' | jq
```

### 6. Inspect the full history

```bash
# Every event in order
curl -s http://localhost:8080/users/42/history | jq

# Compact version timeline (version, event type, timestamp, actor)
curl -s http://localhost:8080/users/42/timeline | jq

# Field-by-field diff between two versions
curl -s 'http://localhost:8080/users/42/diff?from=1&to=3' | jq
```

Search events across aggregates:

```bash
curl -s 'http://localhost:8080/events?aggregate_type=user&aggregate_id=42' | jq
```

### 7. Scrub through versions in the UI

1. Open [http://localhost:8080/ui/](http://localhost:8080/ui/)
2. Set **Entity Type** to `users` and **Entity ID** to `42`
3. Click **Load Timeline**, then drag the version slider or press ▶ to animate through every change
4. Use **Diff v1 → current** to see what changed between the first and latest version

The UI calls the same temporal endpoints (`?version=`, `/timeline`, `/diff`) under the hood.

## API overview

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/{entity}/{id}` | Current state |
| `GET` | `/{entity}/{id}?at=` | State at RFC 3339 timestamp |
| `GET` | `/{entity}/{id}?version=` | State at version |
| `GET` | `/{entity}/{id}/history` | Full event log |
| `GET` | `/{entity}/{id}/timeline` | Version timeline |
| `GET` | `/{entity}/{id}/diff?from=&to=` | Field diff (version or timestamp) |
| `POST` | `/{entity}/{id}/replay` | Explicit replay |
| `GET` | `/events` | Search events |

Entities: `users`, `products`, `orders`.

Full reference: [docs/architecture/REST_API.md](docs/architecture/REST_API.md)

## Local development (without Docker)

```bash
# Start dependencies yourself, then:
make run

# Or build and run manually:
make build
DATABASE_URL="postgres://atm:atm@localhost:5432/atm?sslmode=disable" \
REDIS_URL="redis://localhost:6379/0" \
NATS_URL="nats://localhost:4222" \
./bin/api
```

```bash
make test    # run tests
make bench   # replay benchmarks
make proto   # regenerate gRPC stubs
```

## Architecture

```
REST :8080 / gRPC :9090
        │
   Command services ──▶ Event store (Postgres)
        │                      │
   Replay engine ◀── Snapshots + Redis cache
        │
   Projections ──▶ Read models
        │
   Outbox ──▶ NATS
```

Go packages live under `internal/` (aggregates, replay, diff, projections, workers). Migrations are in `migrations/`.

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/README.md](docs/README.md) | Documentation index |
| [docs/PRD.md](docs/PRD.md) | Product requirements |
| [docs/architecture/SYSTEM_ARCHITECTURE.md](docs/architecture/SYSTEM_ARCHITECTURE.md) | Package layout and data flow |
| [docs/adr/](docs/adr/) | Architecture decision records |

## Tech stack

Go · PostgreSQL 16 · Redis 7 · NATS · gRPC · JWT RBAC · Prometheus · OpenTelemetry

## License

See repository license file if present.
