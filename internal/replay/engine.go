package replay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/eventstore"
	"github.com/api-time-machine/api_time_machine/internal/order"
	"github.com/api-time-machine/api_time_machine/internal/product"
	"github.com/api-time-machine/api_time_machine/internal/snapshot"
	"github.com/api-time-machine/api_time_machine/internal/user"
)

type Aggregate interface {
	Apply(event domain.StoredEvent) error
	UnmarshalSnapshot(data json.RawMessage) error
	currentVersion() int64
	reset(id string)
}

type userAgg struct{ *user.User }

func (u *userAgg) currentVersion() int64 { return u.Version }
func (u *userAgg) reset(id string)       { *u = userAgg{user.New(id)} }

type productAgg struct{ *product.Product }

func (p *productAgg) currentVersion() int64 { return p.Version }
func (p *productAgg) reset(id string)       { *p = productAgg{product.New(id)} }

type orderAgg struct{ *order.Order }

func (o *orderAgg) currentVersion() int64 { return o.Version }
func (o *orderAgg) reset(id string)       { *o = orderAgg{order.New(id)} }

type Engine struct {
	events    *eventstore.Store
	snapshots *snapshot.Store
}

func New(events *eventstore.Store, snapshots *snapshot.Store) *Engine {
	return &Engine{events: events, snapshots: snapshots}
}

func (e *Engine) ReplayUser(ctx context.Context, id string, query domain.ReplayQuery) (*user.User, error) {
	agg := &userAgg{user.New(id)}
	if err := e.replay(ctx, user.AggregateType, id, query, agg); err != nil {
		return nil, err
	}
	return agg.User, nil
}

func (e *Engine) ReplayProduct(ctx context.Context, id string, query domain.ReplayQuery) (*product.Product, error) {
	agg := &productAgg{product.New(id)}
	if err := e.replay(ctx, product.AggregateType, id, query, agg); err != nil {
		return nil, err
	}
	return agg.Product, nil
}

func (e *Engine) ReplayOrder(ctx context.Context, id string, query domain.ReplayQuery) (*order.Order, error) {
	agg := &orderAgg{order.New(id)}
	if err := e.replay(ctx, order.AggregateType, id, query, agg); err != nil {
		return nil, err
	}
	return agg.Order, nil
}

func (e *Engine) replay(ctx context.Context, aggregateType, id string, query domain.ReplayQuery, agg Aggregate) error {
	snap, err := e.snapshots.Latest(ctx, aggregateType, id)
	if err != nil {
		return err
	}

	startVersion := int64(0)
	if snap != nil {
		useSnapshot := true
		if query.Version != nil && *query.Version < snap.Version {
			useSnapshot = false
		}
		if useSnapshot {
			if err := agg.UnmarshalSnapshot(snap.State); err != nil {
				return fmt.Errorf("unmarshal snapshot: %w", err)
			}
			startVersion = snap.Version
		} else {
			agg.reset(id)
		}
	}

	events, err := e.events.LoadEventsUntil(ctx, aggregateType, id, query)
	if err != nil {
		return err
	}

	for _, ev := range events {
		if ev.Version <= startVersion {
			continue
		}
		if err := agg.Apply(ev); err != nil {
			return fmt.Errorf("apply event %d: %w", ev.Version, err)
		}
	}

	if agg.currentVersion() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
