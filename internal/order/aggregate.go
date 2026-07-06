package order

import (
	"encoding/json"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
)

const AggregateType = domain.AggregateOrder

const (
	EventCreated   = "OrderCreated"
	EventUpdated   = "OrderUpdated"
	EventCancelled = "OrderCancelled"
	EventItemAdded = "OrderItemAdded"
	EventItemRemoved = "OrderItemRemoved"
)

type Item struct {
	ProductID  string `json:"product_id"`
	Quantity   int    `json:"quantity"`
	PriceCents int64  `json:"price_cents"`
}

type Order struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Status     string    `json:"status"`
	TotalCents int64     `json:"total_cents"`
	Currency   string    `json:"currency"`
	Items      []Item    `json:"items"`
	Cancelled  bool      `json:"cancelled,omitempty"`
	Version    int64     `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreatedPayload struct {
	UserID     string `json:"user_id"`
	Currency   string `json:"currency"`
	TotalCents int64  `json:"total_cents"`
	Items      []Item `json:"items"`
}

type UpdatedPayload struct {
	Status *string `json:"status,omitempty"`
}

type ItemAddedPayload struct {
	Item Item `json:"item"`
}

type ItemRemovedPayload struct {
	ProductID string `json:"product_id"`
}

func New(id string) *Order {
	return &Order{ID: id, Currency: "USD", Status: "pending", Items: []Item{}}
}

func (o *Order) Apply(event domain.StoredEvent) error {
	o.Version = event.Version
	o.UpdatedAt = event.CreatedAt

	switch event.EventType {
	case EventCreated:
		var payload CreatedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		o.UserID = payload.UserID
		o.TotalCents = payload.TotalCents
		if payload.Currency != "" {
			o.Currency = payload.Currency
		}
		o.Items = payload.Items
		o.Status = "pending"
		o.CreatedAt = event.CreatedAt
	case EventUpdated:
		var payload UpdatedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		if payload.Status != nil {
			o.Status = *payload.Status
		}
	case EventItemAdded:
		var payload ItemAddedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		o.Items = append(o.Items, payload.Item)
		o.recalcTotal()
	case EventItemRemoved:
		var payload ItemRemovedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		filtered := o.Items[:0]
		for _, item := range o.Items {
			if item.ProductID != payload.ProductID {
				filtered = append(filtered, item)
			}
		}
		o.Items = filtered
		o.recalcTotal()
	case EventCancelled:
		o.Cancelled = true
		o.Status = "cancelled"
	}
	return nil
}

func (o *Order) recalcTotal() {
	var total int64
	for _, item := range o.Items {
		total += item.PriceCents * int64(item.Quantity)
	}
	o.TotalCents = total
}

func (o *Order) MarshalSnapshot() (json.RawMessage, error) {
	return json.Marshal(o)
}

func (o *Order) UnmarshalSnapshot(data json.RawMessage) error {
	return json.Unmarshal(data, o)
}

func (o *Order) ToMap() map[string]any {
	return map[string]any{
		"id":          o.ID,
		"user_id":     o.UserID,
		"status":      o.Status,
		"total_cents": o.TotalCents,
		"currency":    o.Currency,
		"items":       o.Items,
		"cancelled":   o.Cancelled,
		"version":     o.Version,
		"created_at":  o.CreatedAt,
		"updated_at":  o.UpdatedAt,
	}
}
