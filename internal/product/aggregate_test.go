package product_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/product"
)

func TestProductReplay(t *testing.T) {
	now := time.Now()
	events := []domain.StoredEvent{
		{Version: 1, EventType: product.EventCreated, Payload: mustJSON(product.CreatedPayload{Name: "Widget", PriceCents: 1000}), CreatedAt: now},
		{Version: 2, EventType: product.EventPriceChanged, Payload: mustJSON(product.PriceChangedPayload{PriceCents: 1500}), CreatedAt: now.Add(time.Hour)},
	}
	p := product.New("p1")
	for _, e := range events {
		if err := p.Apply(e); err != nil {
			t.Fatal(err)
		}
	}
	if p.PriceCents != 1500 {
		t.Fatalf("expected 1500, got %d", p.PriceCents)
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
