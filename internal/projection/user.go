package projection

import (
	"context"
	"errors"
	"fmt"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/user"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserProjector struct {
	pool *pgxpool.Pool
}

func NewUserProjector(pool *pgxpool.Pool) *UserProjector {
	return &UserProjector{pool: pool}
}

func (p *UserProjector) Apply(ctx context.Context, tx pgx.Tx, event domain.StoredEvent) error {
	u := user.New(event.AggregateID)
	if err := u.Apply(event); err != nil {
		return fmt.Errorf("apply event to user: %w", err)
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO users (id, name, email, version, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			email = EXCLUDED.email,
			version = EXCLUDED.version,
			deleted = EXCLUDED.deleted,
			updated_at = EXCLUDED.updated_at
	`, u.ID, u.Name, u.Email, u.Version, u.Deleted, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert user projection: %w", err)
	}
	return nil
}

func (p *UserProjector) GetCurrent(ctx context.Context, id string) (*user.User, error) {
	var u user.User
	err := p.pool.QueryRow(ctx, `
		SELECT id, name, email, version, deleted, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Name, &u.Email, &u.Version, &u.Deleted, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}
