package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/eventstore"
	"github.com/api-time-machine/api_time_machine/internal/replay"
	"github.com/api-time-machine/api_time_machine/internal/snapshot"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

type OutboxPublisher struct {
	pool   *pgxpool.Pool
	nc     *nats.Conn
	logger *slog.Logger
	topic  string
}

func NewOutboxPublisher(pool *pgxpool.Pool, nc *nats.Conn, logger *slog.Logger, topic string) *OutboxPublisher {
	return &OutboxPublisher{pool: pool, nc: nc, logger: logger, topic: topic}
}

func (w *OutboxPublisher) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.publishBatch(ctx)
		}
	}
}

func (w *OutboxPublisher) publishBatch(ctx context.Context) {
	if w.nc == nil {
		return
	}

	rows, err := w.pool.Query(ctx, `
		SELECT id, payload FROM outbox WHERE published = FALSE ORDER BY id LIMIT 100
	`)
	if err != nil {
		w.logger.Error("outbox query", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var payload []byte
		if err := rows.Scan(&id, &payload); err != nil {
			continue
		}
		if err := w.nc.Publish(w.topic, payload); err != nil {
			w.logger.Error("nats publish", "error", err)
			return
		}
		_, _ = w.pool.Exec(ctx, `UPDATE outbox SET published = TRUE WHERE id = $1`, id)
	}
}

type SnapshotBuilder struct {
	events    *eventstore.Store
	snapshots *snapshot.Store
	replay    *replay.Engine
	logger    *slog.Logger
	interval  time.Duration
}

func NewSnapshotBuilder(events *eventstore.Store, snapshots *snapshot.Store, replay *replay.Engine, logger *slog.Logger) *SnapshotBuilder {
	return &SnapshotBuilder{events: events, snapshots: snapshots, replay: replay, logger: logger, interval: 30 * time.Second}
}

func (w *SnapshotBuilder) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.buildSnapshots(ctx)
		}
	}
}

func (w *SnapshotBuilder) buildSnapshots(ctx context.Context) {
	for _, aggType := range []string{"user", "product", "order"} {
		rows, err := w.events.Pool().Query(ctx, `
			SELECT DISTINCT aggregate_id FROM events
			WHERE aggregate_type = $1
			AND aggregate_id NOT IN (
				SELECT aggregate_id FROM snapshots WHERE aggregate_type = $1
				AND version >= (SELECT MAX(version) FROM events e WHERE e.aggregate_type = $1 AND e.aggregate_id = snapshots.aggregate_id)
			)
			LIMIT 20
		`, aggType)
		if err != nil {
			continue
		}

		var ids []string
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				ids = append(ids, id)
			}
		}
		rows.Close()

		for _, id := range ids {
			w.snapshotOne(ctx, aggType, id)
		}
	}
}

func (w *SnapshotBuilder) snapshotOne(ctx context.Context, aggType, id string) {
	var state json.RawMessage
	var version int64

	switch aggType {
	case "user":
		u, err := w.replay.ReplayUser(ctx, id, domain.ReplayQuery{})
		if err != nil {
			return
		}
		state, _ = u.MarshalSnapshot()
		version = u.Version
	case "product":
		p, err := w.replay.ReplayProduct(ctx, id, domain.ReplayQuery{})
		if err != nil {
			return
		}
		state, _ = p.MarshalSnapshot()
		version = p.Version
	case "order":
		o, err := w.replay.ReplayOrder(ctx, id, domain.ReplayQuery{})
		if err != nil {
			return
		}
		state, _ = o.MarshalSnapshot()
		version = o.Version
	default:
		return
	}

	if err := w.snapshots.Save(ctx, aggType, id, version, state); err != nil {
		w.logger.Error("snapshot save", "aggregate", aggType, "id", id, "error", err)
	}
}
