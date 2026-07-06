package order

import (
	"context"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/platform"
)

type ReadModel interface {
	GetCurrent(ctx context.Context, id string) (*Order, error)
}

type Service struct {
	cmds      *platform.CommandService
	projector ReadModel
}

func NewService(cmds *platform.CommandService, projector ReadModel) *Service {
	return &Service{cmds: cmds, projector: projector}
}

type CreateCommand struct {
	ID       string
	UserID   string
	Items    []Item
	Currency string
	Metadata domain.EventMetadata
}

type UpdateCommand struct {
	ID              string
	Status          *string
	ExpectedVersion *int64
	Metadata        domain.EventMetadata
}

type CancelCommand struct {
	ID              string
	ExpectedVersion *int64
	Metadata        domain.EventMetadata
}

type AddItemCommand struct {
	ID              string
	Item            Item
	ExpectedVersion *int64
	Metadata        domain.EventMetadata
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*Order, error) {
	currency := cmd.Currency
	if currency == "" {
		currency = "USD"
	}
	var total int64
	for _, item := range cmd.Items {
		total += item.PriceCents * int64(item.Quantity)
	}
	_, err := s.cmds.Append(ctx, platform.AppendRequest{
		ID: cmd.ID, EventType: EventCreated, IsCreate: true, Metadata: cmd.Metadata,
		Payload: CreatedPayload{UserID: cmd.UserID, Currency: currency, TotalCents: total, Items: cmd.Items},
	})
	if err != nil {
		return nil, err
	}
	return s.projector.GetCurrent(ctx, cmd.ID)
}

func (s *Service) Update(ctx context.Context, cmd UpdateCommand) (*Order, error) {
	_, err := s.cmds.Append(ctx, platform.AppendRequest{
		ID: cmd.ID, ExpectedVersion: cmd.ExpectedVersion, EventType: EventUpdated, Metadata: cmd.Metadata,
		Payload: UpdatedPayload{Status: cmd.Status},
	})
	if err != nil {
		return nil, err
	}
	return s.projector.GetCurrent(ctx, cmd.ID)
}

func (s *Service) Cancel(ctx context.Context, cmd CancelCommand) (*Order, error) {
	_, err := s.cmds.Append(ctx, platform.AppendRequest{
		ID: cmd.ID, ExpectedVersion: cmd.ExpectedVersion, EventType: EventCancelled, Metadata: cmd.Metadata,
		Payload: map[string]any{},
	})
	if err != nil {
		return nil, err
	}
	return s.projector.GetCurrent(ctx, cmd.ID)
}

func (s *Service) AddItem(ctx context.Context, cmd AddItemCommand) (*Order, error) {
	_, err := s.cmds.Append(ctx, platform.AppendRequest{
		ID: cmd.ID, ExpectedVersion: cmd.ExpectedVersion, EventType: EventItemAdded, Metadata: cmd.Metadata,
		Payload: ItemAddedPayload{Item: cmd.Item},
	})
	if err != nil {
		return nil, err
	}
	return s.projector.GetCurrent(ctx, cmd.ID)
}

func (s *Service) GetCurrent(ctx context.Context, id string) (*Order, error) {
	return s.projector.GetCurrent(ctx, id)
}
