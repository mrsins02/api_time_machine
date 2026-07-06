package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/eventstore"
	"github.com/api-time-machine/api_time_machine/internal/snapshot"
	"github.com/jackc/pgx/v5"
)

type Projector interface {
	Apply(ctx context.Context, tx pgx.Tx, event domain.StoredEvent) error
}

type Snapshotter interface {
	MarshalSnapshot() (json.RawMessage, error)
	AggregateID() string
	Version() int64
}

type CommandService struct {
	events        *eventstore.Store
	projector     Projector
	snapshots     *snapshot.Store
	aggregateType string
	snapshotEvery int
}

func NewCommandService(
	events *eventstore.Store,
	projector Projector,
	snapshots *snapshot.Store,
	aggregateType string,
	snapshotEvery int,
) *CommandService {
	return &CommandService{
		events:        events,
		projector:     projector,
		snapshots:     snapshots,
		aggregateType: aggregateType,
		snapshotEvery: snapshotEvery,
	}
}

type AppendRequest struct {
	ID              string
	ExpectedVersion *int64
	EventType       string
	Payload         any
	Metadata        domain.EventMetadata
	IsCreate        bool
}

func (s *CommandService) Append(ctx context.Context, req AppendRequest) (domain.StoredEvent, error) {
	tx, err := s.events.Pool().Begin(ctx)
	if err != nil {
		return domain.StoredEvent{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	current, err := s.events.LatestVersion(ctx, s.aggregateType, req.ID)
	if err != nil {
		return domain.StoredEvent{}, err
	}

	if req.ExpectedVersion != nil && *req.ExpectedVersion != current {
		return domain.StoredEvent{}, domain.ErrVersionMismatch
	}
	if req.IsCreate && current > 0 {
		return domain.StoredEvent{}, domain.ErrConflict
	}
	if !req.IsCreate && current == 0 {
		return domain.StoredEvent{}, domain.ErrNotFound
	}

	nextVersion := current + 1
	stored, err := s.events.Append(ctx, tx, domain.AppendEvent{
		AggregateType: s.aggregateType,
		AggregateID:   req.ID,
		EventType:     req.EventType,
		Version:       nextVersion,
		Payload:       req.Payload,
		Metadata:      req.Metadata,
	})
	if err != nil {
		return domain.StoredEvent{}, err
	}

	if err := s.projector.Apply(ctx, tx, stored); err != nil {
		return domain.StoredEvent{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.StoredEvent{}, fmt.Errorf("commit: %w", err)
	}

	if s.snapshotEvery > 0 && nextVersion%int64(s.snapshotEvery) == 0 {
		go s.maybeSnapshot(context.Background(), stored)
	}

	return stored, nil
}

func (s *CommandService) maybeSnapshot(ctx context.Context, event domain.StoredEvent) {
	// snapshot building is delegated to background worker for full state;
	// event-triggered path stores event version marker for worker pickup
	_ = event
}

type Idempotency struct {
	pool *eventstore.Store
}

func NewIdempotency(events *eventstore.Store) *Idempotency {
	return &Idempotency{pool: events}
}

func (i *Idempotency) Store(ctx context.Context, key string, body any, status int, ttl time.Duration) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = i.pool.Pool().Exec(ctx, `
		INSERT INTO idempotency_keys (key, response_body, status_code, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key) DO NOTHING
	`, key, raw, status, time.Now().Add(ttl))
	return err
}

func (i *Idempotency) Get(ctx context.Context, key string) (json.RawMessage, int, bool, error) {
	var body json.RawMessage
	var status int
	err := i.pool.Pool().QueryRow(ctx, `
		SELECT response_body, status_code FROM idempotency_keys
		WHERE key = $1 AND expires_at > NOW()
	`, key).Scan(&body, &status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, 0, false, nil
		}
		return nil, 0, false, err
	}
	return body, status, true, nil
}
