package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/replay"
	"github.com/api-time-machine/api_time_machine/internal/user"
	"github.com/redis/go-redis/v9"
)

type ReplayCache struct {
	client *redis.Client
	inner  *replay.Engine
	ttl    time.Duration
}

func NewReplayCache(client *redis.Client, inner *replay.Engine, ttl time.Duration) *ReplayCache {
	return &ReplayCache{client: client, inner: inner, ttl: ttl}
}

func (c *ReplayCache) ReplayUser(ctx context.Context, id string, query domain.ReplayQuery) (*user.User, error) {
	if c.client == nil {
		return c.inner.ReplayUser(ctx, id, query)
	}

	key := cacheKey("user", id, query)
	if raw, err := c.client.Get(ctx, key).Bytes(); err == nil {
		var u user.User
		if json.Unmarshal(raw, &u) == nil {
			return &u, nil
		}
	}

	u, err := c.inner.ReplayUser(ctx, id, query)
	if err != nil {
		return nil, err
	}

	if raw, err := json.Marshal(u); err == nil {
		_ = c.client.Set(ctx, key, raw, c.ttl).Err()
	}
	return u, nil
}

func (c *ReplayCache) Inner() *replay.Engine {
	return c.inner
}

func cacheKey(aggregateType, id string, query domain.ReplayQuery) string {
	if query.Version != nil {
		return fmt.Sprintf("replay:%s:%s:v:%d", aggregateType, id, *query.Version)
	}
	if query.At != nil {
		return fmt.Sprintf("replay:%s:%s:at:%d", aggregateType, id, query.At.UnixNano())
	}
	return fmt.Sprintf("replay:%s:%s:current", aggregateType, id)
}

func NewRedisClient(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	return client, nil
}

func Ping(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return nil
	}
	return client.Ping(ctx).Err()
}
