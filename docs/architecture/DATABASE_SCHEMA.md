# Database Schema

PostgreSQL 16. All timestamps are `TIMESTAMPTZ`. Migrations live in `migrations/`.

## events (append-only)

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | Monotonic event ID |
| aggregate_type | TEXT | `user`, `product`, `order` |
| aggregate_id | TEXT | Entity identifier |
| event_type | TEXT | e.g. `UserCreated` |
| version | BIGINT | Per-aggregate sequence |
| payload | JSONB | Event data |
| metadata | JSONB | actor, correlation_id, trace_id |
| created_at | TIMESTAMPTZ | Wall-clock time of append |

**Constraints:** `UNIQUE (aggregate_type, aggregate_id, version)`

**Indexes:** aggregate lookup, created_at, event_type, metadata actor/correlation

## snapshots

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| aggregate_type | TEXT | |
| aggregate_id | TEXT | |
| version | BIGINT | Snapshot at this version |
| state | JSONB | Serialized aggregate |
| created_at | TIMESTAMPTZ | |

**Constraints:** `UNIQUE (aggregate_type, aggregate_id, version)`

## Read Models

### users

`id`, `name`, `email`, `version`, `deleted`, `created_at`, `updated_at`

### products

`id`, `name`, `description`, `price_cents`, `currency`, `version`, `deleted`, `created_at`, `updated_at`

### orders

`id`, `user_id`, `status`, `total_cents`, `currency`, `items` (JSONB), `version`, `cancelled`, `created_at`, `updated_at`

## outbox (transactional outbox)

| Column | Type | Notes |
|--------|------|-------|
| id | BIGSERIAL PK | |
| aggregate_type | TEXT | |
| aggregate_id | TEXT | |
| event_id | BIGINT FK → events | |
| payload | JSONB | Published message body |
| published | BOOLEAN | Default false |
| created_at | TIMESTAMPTZ | |

**Index:** partial on `published = FALSE`

## idempotency_keys

| Column | Type | Notes |
|--------|------|-------|
| key | TEXT PK | `Idempotency-Key` header value |
| response_body | JSONB | Cached response |
| status_code | INT | HTTP status |
| expires_at | TIMESTAMPTZ | TTL enforcement |

## schema_migrations

Tracks applied SQL migration files.
