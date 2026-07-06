package projection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/order"
	"github.com/api-time-machine/api_time_machine/internal/product"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductProjector struct {
	pool *pgxpool.Pool
}

func NewProductProjector(pool *pgxpool.Pool) *ProductProjector {
	return &ProductProjector{pool: pool}
}

func (p *ProductProjector) Apply(ctx context.Context, tx pgx.Tx, event domain.StoredEvent) error {
	prod := product.New(event.AggregateID)
	if err := prod.Apply(event); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO products (id, name, description, price_cents, currency, version, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			price_cents = EXCLUDED.price_cents,
			currency = EXCLUDED.currency,
			version = EXCLUDED.version,
			deleted = EXCLUDED.deleted,
			updated_at = EXCLUDED.updated_at
	`, prod.ID, prod.Name, prod.Description, prod.PriceCents, prod.Currency, prod.Version, prod.Deleted, prod.CreatedAt, prod.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert product projection: %w", err)
	}
	return nil
}

func (p *ProductProjector) GetCurrent(ctx context.Context, id string) (*product.Product, error) {
	var prod product.Product
	err := p.pool.QueryRow(ctx, `
		SELECT id, name, description, price_cents, currency, version, deleted, created_at, updated_at
		FROM products WHERE id = $1
	`, id).Scan(&prod.ID, &prod.Name, &prod.Description, &prod.PriceCents, &prod.Currency, &prod.Version, &prod.Deleted, &prod.CreatedAt, &prod.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get product: %w", err)
	}
	return &prod, nil
}

type OrderProjector struct {
	pool *pgxpool.Pool
}

func NewOrderProjector(pool *pgxpool.Pool) *OrderProjector {
	return &OrderProjector{pool: pool}
}

func (p *OrderProjector) Apply(ctx context.Context, tx pgx.Tx, event domain.StoredEvent) error {
	o := order.New(event.AggregateID)
	if err := o.Apply(event); err != nil {
		return err
	}
	items, err := json.Marshal(o.Items)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, user_id, status, total_cents, currency, items, version, cancelled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			status = EXCLUDED.status,
			total_cents = EXCLUDED.total_cents,
			currency = EXCLUDED.currency,
			items = EXCLUDED.items,
			version = EXCLUDED.version,
			cancelled = EXCLUDED.cancelled,
			updated_at = EXCLUDED.updated_at
	`, o.ID, o.UserID, o.Status, o.TotalCents, o.Currency, items, o.Version, o.Cancelled, o.CreatedAt, o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert order projection: %w", err)
	}
	return nil
}

func (p *OrderProjector) GetCurrent(ctx context.Context, id string) (*order.Order, error) {
	var o order.Order
	var items []byte
	err := p.pool.QueryRow(ctx, `
		SELECT id, user_id, status, total_cents, currency, items, version, cancelled, created_at, updated_at
		FROM orders WHERE id = $1
	`, id).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalCents, &o.Currency, &items, &o.Version, &o.Cancelled, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	if err := json.Unmarshal(items, &o.Items); err != nil {
		return nil, err
	}
	return &o, nil
}
