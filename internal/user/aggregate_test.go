package user_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/user"
)

func TestUserReplayFromEvents(t *testing.T) {
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	events := []domain.StoredEvent{
		mkEvent(1, user.EventCreated, user.CreatedPayload{Name: "Ali", Email: "old@mail.com"}, now),
		mkEvent(2, user.EventEmailChanged, user.EmailChangedPayload{Email: "ali@gmail.com"}, now.Add(time.Hour)),
		mkEvent(3, user.EventUpdated, user.UpdatedPayload{Name: strPtr("Ali B")}, now.Add(3*time.Hour)),
	}

	u := user.New("42")
	for _, e := range events {
		if err := u.Apply(e); err != nil {
			t.Fatal(err)
		}
	}

	if u.Email != "ali@gmail.com" {
		t.Fatalf("expected current email, got %s", u.Email)
	}

	// replay to version 1
	past := user.New("42")
	for _, e := range events {
		if e.Version > 1 {
			break
		}
		if err := past.Apply(e); err != nil {
			t.Fatal(err)
		}
	}
	if past.Email != "old@mail.com" {
		t.Fatalf("expected old email at v1, got %s", past.Email)
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	u := &user.User{
		ID: "42", Name: "Ali", Email: "a@b.com", Version: 5,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	raw, err := u.MarshalSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored := user.New("42")
	if err := restored.UnmarshalSnapshot(raw); err != nil {
		t.Fatal(err)
	}
	if restored.Version != 5 || restored.Email != "a@b.com" {
		t.Fatalf("snapshot mismatch: %+v", restored)
	}
}

func mkEvent(version int64, eventType string, payload any, at time.Time) domain.StoredEvent {
	raw, _ := json.Marshal(payload)
	return domain.StoredEvent{
		Version: version, EventType: eventType, Payload: raw, CreatedAt: at,
	}
}

func strPtr(s string) *string { return &s }
