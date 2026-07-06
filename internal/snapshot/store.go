package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Latest(ctx context.Context, aggregateType, aggregateID string) (*domain.Snapshot, error) {
	var snap domain.Snapshot
	err := s.pool.QueryRow(ctx, `
		SELECT id, aggregate_type, aggregate_id, version, state, created_at
		FROM snapshots
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version DESC
		LIMIT 1
	`, aggregateType, aggregateID).Scan(
		&snap.ID, &snap.AggregateType, &snap.AggregateID,
		&snap.Version, &snap.State, &snap.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("latest snapshot: %w", err)
	}
	return &snap, nil
}

func (s *Store) Save(ctx context.Context, aggregateType, aggregateID string, version int64, state json.RawMessage) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO snapshots (aggregate_type, aggregate_id, version, state)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (aggregate_type, aggregate_id, version) DO UPDATE SET state = EXCLUDED.state
	`, aggregateType, aggregateID, version, state)
	if err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}
