package product

import (
	"encoding/json"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
)

const AggregateType = domain.AggregateProduct

const (
	EventCreated      = "ProductCreated"
	EventUpdated      = "ProductUpdated"
	EventDeleted      = "ProductDeleted"
	EventPriceChanged = "PriceChanged"
)

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCents  int64     `json:"price_cents"`
	Currency    string    `json:"currency"`
	Deleted     bool      `json:"deleted,omitempty"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreatedPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Currency    string `json:"currency"`
}

type UpdatedPayload struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	PriceCents  *int64  `json:"price_cents,omitempty"`
	Currency    *string `json:"currency,omitempty"`
}

type PriceChangedPayload struct {
	PriceCents int64  `json:"price_cents"`
	Currency   string `json:"currency,omitempty"`
}

func New(id string) *Product {
	return &Product{ID: id, Currency: "USD"}
}

func (p *Product) Apply(event domain.StoredEvent) error {
	p.Version = event.Version
	p.UpdatedAt = event.CreatedAt

	switch event.EventType {
	case EventCreated:
		var payload CreatedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		p.Name = payload.Name
		p.Description = payload.Description
		p.PriceCents = payload.PriceCents
		if payload.Currency != "" {
			p.Currency = payload.Currency
		}
		p.CreatedAt = event.CreatedAt
		p.Deleted = false
	case EventUpdated:
		var payload UpdatedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		if payload.Name != nil {
			p.Name = *payload.Name
		}
		if payload.Description != nil {
			p.Description = *payload.Description
		}
		if payload.PriceCents != nil {
			p.PriceCents = *payload.PriceCents
		}
		if payload.Currency != nil {
			p.Currency = *payload.Currency
		}
	case EventPriceChanged:
		var payload PriceChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		p.PriceCents = payload.PriceCents
		if payload.Currency != "" {
			p.Currency = payload.Currency
		}
	case EventDeleted:
		p.Deleted = true
	}
	return nil
}

func (p *Product) MarshalSnapshot() (json.RawMessage, error) {
	return json.Marshal(p)
}

func (p *Product) UnmarshalSnapshot(data json.RawMessage) error {
	return json.Unmarshal(data, p)
}

func (p *Product) ToMap() map[string]any {
	return map[string]any{
		"id":          p.ID,
		"name":        p.Name,
		"description": p.Description,
		"price_cents": p.PriceCents,
		"currency":    p.Currency,
		"deleted":     p.Deleted,
		"version":     p.Version,
		"created_at":  p.CreatedAt,
		"updated_at":  p.UpdatedAt,
	}
}
