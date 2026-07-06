CREATE TABLE IF NOT EXISTS products (
    id          TEXT        PRIMARY KEY,
    name        TEXT        NOT NULL DEFAULT '',
    description TEXT        NOT NULL DEFAULT '',
    price_cents BIGINT      NOT NULL DEFAULT 0,
    currency    TEXT        NOT NULL DEFAULT 'USD',
    version     BIGINT      NOT NULL DEFAULT 0,
    deleted     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_name ON products (name);

CREATE TABLE IF NOT EXISTS orders (
    id          TEXT        PRIMARY KEY,
    user_id     TEXT        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'pending',
    total_cents BIGINT      NOT NULL DEFAULT 0,
    currency    TEXT        NOT NULL DEFAULT 'USD',
    items       JSONB       NOT NULL DEFAULT '[]',
    version     BIGINT      NOT NULL DEFAULT 0,
    cancelled   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
