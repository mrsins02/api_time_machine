package replay_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/user"
)

func BenchmarkUserReplayApply(b *testing.B) {
	events := make([]domain.StoredEvent, 100)
	now := time.Now()
	for i := range events {
		raw, _ := json.Marshal(user.EmailChangedPayload{Email: "a@b.com"})
		events[i] = domain.StoredEvent{
			Version: int64(i + 1), EventType: user.EventEmailChanged,
			Payload: raw, CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		u := user.New("42")
		for _, e := range events {
			_ = u.Apply(e)
		}
	}
	_ = context.Background()
}
