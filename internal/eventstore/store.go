package eventstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *Store) Append(ctx context.Context, tx pgx.Tx, event domain.AppendEvent) (domain.StoredEvent, error) {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return domain.StoredEvent{}, fmt.Errorf("marshal payload: %w", err)
	}
	meta, err := json.Marshal(event.Metadata)
	if err != nil {
		return domain.StoredEvent{}, fmt.Errorf("marshal metadata: %w", err)
	}

	var stored domain.StoredEvent
	var metaRaw []byte
	err = tx.QueryRow(ctx, `
		INSERT INTO events (aggregate_type, aggregate_id, event_type, version, payload, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, aggregate_type, aggregate_id, event_type, version, payload, metadata, created_at
	`, event.AggregateType, event.AggregateID, event.EventType, event.Version, payload, meta).Scan(
		&stored.ID,
		&stored.AggregateType,
		&stored.AggregateID,
		&stored.EventType,
		&stored.Version,
		&stored.Payload,
		&metaRaw,
		&stored.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.StoredEvent{}, domain.ErrConflict
		}
		return domain.StoredEvent{}, fmt.Errorf("insert event: %w", err)
	}
	if err := json.Unmarshal(metaRaw, &stored.Metadata); err != nil {
		return domain.StoredEvent{}, fmt.Errorf("unmarshal metadata: %w", err)
	}

	outboxPayload, _ := json.Marshal(map[string]any{
		"event_id":        stored.ID,
		"aggregate_type":  stored.AggregateType,
		"aggregate_id":    stored.AggregateID,
		"event_type":      stored.EventType,
		"version":         stored.Version,
		"payload":         json.RawMessage(payload),
	})
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox (aggregate_type, aggregate_id, event_id, payload)
		VALUES ($1, $2, $3, $4)
	`, stored.AggregateType, stored.AggregateID, stored.ID, outboxPayload); err != nil {
		return domain.StoredEvent{}, fmt.Errorf("insert outbox: %w", err)
	}

	return stored, nil
}

func (s *Store) LoadEvents(ctx context.Context, aggregateType, aggregateID string, afterVersion int64) ([]domain.StoredEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, version, payload, metadata, created_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version > $3
		ORDER BY version ASC
	`, aggregateType, aggregateID, afterVersion)
	if err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (s *Store) LoadEventsUntil(ctx context.Context, aggregateType, aggregateID string, query domain.ReplayQuery) ([]domain.StoredEvent, error) {
	base := `
		SELECT id, aggregate_type, aggregate_id, event_type, version, payload, metadata, created_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
	`
	args := []any{aggregateType, aggregateID}
	argN := 3

	if query.Version != nil {
		base += fmt.Sprintf(" AND version <= $%d", argN)
		args = append(args, *query.Version)
		argN++
	}
	if query.At != nil {
		base += fmt.Sprintf(" AND created_at <= $%d", argN)
		args = append(args, *query.At)
		argN++
	}
	if query.UntilID != nil {
		base += fmt.Sprintf(" AND id <= $%d", argN)
		args = append(args, *query.UntilID)
	}

	base += " ORDER BY version ASC"

	rows, err := s.pool.Query(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("load events until: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (s *Store) LatestVersion(ctx context.Context, aggregateType, aggregateID string) (int64, error) {
	var version int64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0)
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
	`, aggregateType, aggregateID).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("latest version: %w", err)
	}
	return version, nil
}

func (s *Store) Timeline(ctx context.Context, aggregateType, aggregateID string) ([]domain.TimelineEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, version, event_type, created_at, metadata
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC
	`, aggregateType, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("timeline: %w", err)
	}
	defer rows.Close()

	var entries []domain.TimelineEntry
	for rows.Next() {
		var e domain.TimelineEntry
		var meta []byte
		if err := rows.Scan(&e.EventID, &e.Version, &e.EventType, &e.Timestamp, &meta); err != nil {
			return nil, err
		}
		var m domain.EventMetadata
		_ = json.Unmarshal(meta, &m)
		e.Actor = m.Actor
		e.Summary = e.EventType
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) History(ctx context.Context, aggregateType, aggregateID string) ([]domain.StoredEvent, error) {
	return s.LoadEventsUntil(ctx, aggregateType, aggregateID, domain.ReplayQuery{})
}

type SearchFilter struct {
	AggregateType string
	AggregateID   string
	EventType     string
	Actor         string
	CorrelationID string
	From          *time.Time
	To            *time.Time
}

func (s *Store) Search(ctx context.Context, filter SearchFilter) ([]domain.StoredEvent, error) {
	q := `
		SELECT id, aggregate_type, aggregate_id, event_type, version, payload, metadata, created_at
		FROM events
		WHERE 1=1
	`
	args := []any{}
	n := 1

	if filter.AggregateType != "" {
		q += fmt.Sprintf(" AND aggregate_type = $%d", n)
		args = append(args, filter.AggregateType)
		n++
	}
	if filter.AggregateID != "" {
		q += fmt.Sprintf(" AND aggregate_id = $%d", n)
		args = append(args, filter.AggregateID)
		n++
	}
	if filter.EventType != "" {
		q += fmt.Sprintf(" AND event_type = $%d", n)
		args = append(args, filter.EventType)
		n++
	}
	if filter.Actor != "" {
		q += fmt.Sprintf(" AND metadata->>'actor' = $%d", n)
		args = append(args, filter.Actor)
		n++
	}
	if filter.CorrelationID != "" {
		q += fmt.Sprintf(" AND metadata->>'correlation_id' = $%d", n)
		args = append(args, filter.CorrelationID)
		n++
	}
	if filter.From != nil {
		q += fmt.Sprintf(" AND created_at >= $%d", n)
		args = append(args, *filter.From)
		n++
	}
	if filter.To != nil {
		q += fmt.Sprintf(" AND created_at <= $%d", n)
		args = append(args, *filter.To)
		n++
	}

	q += " ORDER BY created_at DESC LIMIT 500"

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search events: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

func scanEvents(rows pgx.Rows) ([]domain.StoredEvent, error) {
	var events []domain.StoredEvent
	for rows.Next() {
		var e domain.StoredEvent
		var meta []byte
		if err := rows.Scan(
			&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType,
			&e.Version, &e.Payload, &meta, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(meta, &e.Metadata); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
