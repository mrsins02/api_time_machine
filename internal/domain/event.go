package domain

import (
	"encoding/json"
	"time"
)

type EventMetadata struct {
	Actor         string `json:"actor,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type StoredEvent struct {
	ID             int64           `json:"id"`
	AggregateType  string          `json:"aggregate_type"`
	AggregateID    string          `json:"aggregate_id"`
	EventType      string          `json:"event_type"`
	Version        int64           `json:"version"`
	Payload        json.RawMessage `json:"payload"`
	Metadata       EventMetadata   `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
}

type Snapshot struct {
	ID            int64           `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	Version       int64           `json:"version"`
	State         json.RawMessage `json:"state"`
	CreatedAt     time.Time       `json:"created_at"`
}

type AppendEvent struct {
	AggregateType string
	AggregateID   string
	EventType     string
	Version       int64
	Payload       any
	Metadata      EventMetadata
}

type ReplayQuery struct {
	At        *time.Time
	Version   *int64
	UntilID   *int64
}

func IsTemporal(q ReplayQuery) bool {
	return q.At != nil || q.Version != nil || q.UntilID != nil
}

type TimelineEntry struct {
	Version   int64     `json:"version"`
	EventType string    `json:"event_type"`
	EventID   int64     `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor,omitempty"`
	Summary   string    `json:"summary,omitempty"`
}

type DiffResult struct {
	Changed []string               `json:"changed"`
	From    map[string]any         `json:"from"`
	To      map[string]any         `json:"to"`
	Fields  map[string]FieldChange `json:"fields,omitempty"`
}

type FieldChange struct {
	From any `json:"from"`
	To   any `json:"to"`
}

var (
	ErrConflict         = errConflict{}
	ErrNotFound         = errNotFound{}
	ErrVersionMismatch  = errVersionMismatch{}
)

type errConflict struct{}
func (errConflict) Error() string { return "aggregate conflict: version already exists" }

type errNotFound struct{}
func (errNotFound) Error() string { return "aggregate not found" }

type errVersionMismatch struct{}
func (errVersionMismatch) Error() string { return "expected version mismatch" }
