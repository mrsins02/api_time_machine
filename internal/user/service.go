package user

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
	GetCurrent(ctx context.Context, id string) (*User, error)
}

type Service struct {
	events    *eventstore.Store
	projector Projector
	snapshots *snapshot.Store
	snapshotN int
}

func NewService(events *eventstore.Store, projector Projector, snapshots *snapshot.Store, snapshotEvery int) *Service {
	return &Service{
		events:    events,
		projector: projector,
		snapshots: snapshots,
		snapshotN: snapshotEvery,
	}
}

type CreateCommand struct {
	ID              string
	Name            string
	Email           string
	ExpectedVersion *int64
	Metadata        domain.EventMetadata
}

type UpdateCommand struct {
	ID              string
	Name            *string
	Email           *string
	Address         *string
	Avatar          *string
	ExpectedVersion *int64
	Metadata        domain.EventMetadata
}

type DeleteCommand struct {
	ID              string
	ExpectedVersion *int64
	Metadata        domain.EventMetadata
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*User, error) {
	return s.append(ctx, cmd.ID, cmd.ExpectedVersion, EventCreated, CreatedPayload{
		Name:  cmd.Name,
		Email: cmd.Email,
	}, cmd.Metadata)
}

func (s *Service) Update(ctx context.Context, cmd UpdateCommand) (*User, error) {
	return s.append(ctx, cmd.ID, cmd.ExpectedVersion, EventUpdated, UpdatedPayload{
		Name:    cmd.Name,
		Email:   cmd.Email,
		Address: cmd.Address,
		Avatar:  cmd.Avatar,
	}, cmd.Metadata)
}

func (s *Service) Delete(ctx context.Context, cmd DeleteCommand) (*User, error) {
	return s.append(ctx, cmd.ID, cmd.ExpectedVersion, EventDeleted, map[string]any{}, cmd.Metadata)
}

func (s *Service) append(ctx context.Context, id string, expected *int64, eventType string, payload any, meta domain.EventMetadata) (*User, error) {
	tx, err := s.events.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	current, err := s.events.LatestVersion(ctx, AggregateType, id)
	if err != nil {
		return nil, err
	}

	if expected != nil && *expected != current {
		return nil, domain.ErrVersionMismatch
	}
	if eventType == EventCreated && current > 0 {
		return nil, domain.ErrConflict
	}
	if eventType != EventCreated && current == 0 {
		return nil, domain.ErrNotFound
	}

	nextVersion := current + 1
	stored, err := s.events.Append(ctx, tx, domain.AppendEvent{
		AggregateType: AggregateType,
		AggregateID:   id,
		EventType:     eventType,
		Version:       nextVersion,
		Payload:       payload,
		Metadata:      meta,
	})
	if err != nil {
		return nil, err
	}

	if err := s.projector.Apply(ctx, tx, stored); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	u, err := s.projector.GetCurrent(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.snapshotN > 0 && nextVersion%int64(s.snapshotN) == 0 {
		go s.maybeSnapshot(context.Background(), u)
	}

	return u, nil
}

func (s *Service) maybeSnapshot(ctx context.Context, u *User) {
	state, err := u.MarshalSnapshot()
	if err != nil {
		return
	}
	_ = s.snapshots.Save(ctx, AggregateType, u.ID, u.Version, state)
}

func (s *Service) StoreIdempotency(ctx context.Context, key string, body any, status int, ttl time.Duration) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = s.events.Pool().Exec(ctx, `
		INSERT INTO idempotency_keys (key, response_body, status_code, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key) DO NOTHING
	`, key, raw, status, time.Now().Add(ttl))
	return err
}

func (s *Service) GetIdempotency(ctx context.Context, key string) (json.RawMessage, int, bool, error) {
	var body json.RawMessage
	var status int
	err := s.events.Pool().QueryRow(ctx, `
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
