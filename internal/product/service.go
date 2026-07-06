package product

import (
	"context"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/platform"
)

type ReadModel interface {
	GetCurrent(ctx context.Context, id string) (*Product, error)
}

type Service struct {
	cmds      *platform.CommandService
	projector ReadModel
}

func NewService(cmds *platform.CommandService, projector ReadModel) *Service {
	return &Service{cmds: cmds, projector: projector}
}

type CreateCommand struct {
	ID          string
	Name        string
	Description string
	PriceCents  int64
	Currency    string
	Metadata    domain.EventMetadata
}

type UpdateCommand struct {
	ID              string
	Name            *string
	Description     *string
	PriceCents      *int64
	Currency        *string
	ExpectedVersion *int64
	Metadata        domain.EventMetadata
}

type DeleteCommand struct {
	ID              string
	ExpectedVersion *int64
	Metadata        domain.EventMetadata
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*Product, error) {
	currency := cmd.Currency
	if currency == "" {
		currency = "USD"
	}
	_, err := s.cmds.Append(ctx, platform.AppendRequest{
		ID: cmd.ID, EventType: EventCreated, IsCreate: true, Metadata: cmd.Metadata,
		Payload: CreatedPayload{Name: cmd.Name, Description: cmd.Description, PriceCents: cmd.PriceCents, Currency: currency},
	})
	if err != nil {
		return nil, err
	}
	return s.projector.GetCurrent(ctx, cmd.ID)
}

func (s *Service) Update(ctx context.Context, cmd UpdateCommand) (*Product, error) {
	_, err := s.cmds.Append(ctx, platform.AppendRequest{
		ID: cmd.ID, ExpectedVersion: cmd.ExpectedVersion, EventType: EventUpdated, Metadata: cmd.Metadata,
		Payload: UpdatedPayload{Name: cmd.Name, Description: cmd.Description, PriceCents: cmd.PriceCents, Currency: cmd.Currency},
	})
	if err != nil {
		return nil, err
	}
	return s.projector.GetCurrent(ctx, cmd.ID)
}

func (s *Service) Delete(ctx context.Context, cmd DeleteCommand) (*Product, error) {
	_, err := s.cmds.Append(ctx, platform.AppendRequest{
		ID: cmd.ID, ExpectedVersion: cmd.ExpectedVersion, EventType: EventDeleted, Metadata: cmd.Metadata,
		Payload: map[string]any{},
	})
	if err != nil {
		return nil, err
	}
	return s.projector.GetCurrent(ctx, cmd.ID)
}

func (s *Service) GetCurrent(ctx context.Context, id string) (*Product, error) {
	return s.projector.GetCurrent(ctx, id)
}
