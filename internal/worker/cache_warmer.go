package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/eventstore"
	"github.com/api-time-machine/api_time_machine/internal/replay"
	"github.com/redis/go-redis/v9"
)

type CacheWarmer struct {
	events *eventstore.Store
	replay *replay.Engine
	redis  *redis.Client
	ttl    time.Duration
	logger *slog.Logger
}

func NewCacheWarmer(events *eventstore.Store, replay *replay.Engine, redis *redis.Client, ttl time.Duration, logger *slog.Logger) *CacheWarmer {
	return &CacheWarmer{events: events, replay: replay, redis: redis, ttl: ttl, logger: logger}
}

func (w *CacheWarmer) Run(ctx context.Context) {
	if w.redis == nil {
		return
	}
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.warm(ctx)
		}
	}
}

func (w *CacheWarmer) warm(ctx context.Context) {
	rows, err := w.events.Pool().Query(ctx, `
		SELECT DISTINCT aggregate_type, aggregate_id
		FROM events
		ORDER BY aggregate_id
		LIMIT 50
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var aggType, id string
		if rows.Scan(&aggType, &id) != nil {
			continue
		}
		switch aggType {
		case "user":
			if u, err := w.replay.ReplayUser(ctx, id, domain.ReplayQuery{}); err == nil {
				_ = u
			}
		}
	}
}
