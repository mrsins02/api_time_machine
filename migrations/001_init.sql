-- Event store: append-only, immutable history
CREATE TABLE IF NOT EXISTS events (
    id              BIGSERIAL PRIMARY KEY,
    aggregate_type  TEXT        NOT NULL,
    aggregate_id    TEXT        NOT NULL,
    event_type      TEXT        NOT NULL,
    version         BIGINT      NOT NULL,
    payload         JSONB       NOT NULL DEFAULT '{}',
    metadata        JSONB       NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT events_aggregate_version_unique UNIQUE (aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_aggregate
    ON events (aggregate_type, aggregate_id, version);

CREATE INDEX IF NOT EXISTS idx_events_created_at
    ON events (aggregate_type, aggregate_id, created_at);

CREATE INDEX IF NOT EXISTS idx_events_type
    ON events (event_type);

CREATE INDEX IF NOT EXISTS idx_events_metadata_actor
    ON events ((metadata->>'actor'));

CREATE INDEX IF NOT EXISTS idx_events_metadata_correlation
    ON events ((metadata->>'correlation_id'));

-- Snapshots for fast replay
CREATE TABLE IF NOT EXISTS snapshots (
    id              BIGSERIAL PRIMARY KEY,
    aggregate_type  TEXT        NOT NULL,
    aggregate_id    TEXT        NOT NULL,
    version         BIGINT      NOT NULL,
    state           JSONB       NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT snapshots_aggregate_version_unique UNIQUE (aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_snapshots_aggregate
    ON snapshots (aggregate_type, aggregate_id, version DESC);

-- CQRS read model: current user projection
CREATE TABLE IF NOT EXISTS users (
    id          TEXT        PRIMARY KEY,
    name        TEXT        NOT NULL DEFAULT '',
    email       TEXT        NOT NULL DEFAULT '',
    version     BIGINT      NOT NULL DEFAULT 0,
    deleted     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- Transactional outbox for async consumers
CREATE TABLE IF NOT EXISTS outbox (
    id              BIGSERIAL PRIMARY KEY,
    aggregate_type  TEXT        NOT NULL,
    aggregate_id    TEXT        NOT NULL,
    event_id        BIGINT      NOT NULL REFERENCES events(id),
    payload         JSONB       NOT NULL,
    published       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
    ON outbox (id) WHERE published = FALSE;

-- Idempotency keys for command deduplication
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key             TEXT        PRIMARY KEY,
    response_body   JSONB       NOT NULL,
    status_code     INT         NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON idempotency_keys (expires_at);
