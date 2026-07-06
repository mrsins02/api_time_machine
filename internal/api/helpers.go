package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/eventstore"
	"github.com/go-chi/chi/v5"
)

type replayQueryParams struct {
	at      *time.Time
	version *int64
}

func (q replayQueryParams) isTemporal() bool {
	return q.at != nil || q.version != nil
}

func (q replayQueryParams) toDomain() domain.ReplayQuery {
	return domain.ReplayQuery{At: q.at, Version: q.version}
}

func parseReplayQuery(r *http.Request) (replayQueryParams, error) {
	q := replayQueryParams{}
	if v := r.URL.Query().Get("at"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return q, err
		}
		q.at = &t
	}
	if v := r.URL.Query().Get("version"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return q, err
		}
		q.version = &n
	}
	return q, nil
}

func parseTimeOrVersionParam(r *http.Request, key string) (domain.ReplayQuery, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return domain.ReplayQuery{}, nil
	}
	if strings.HasPrefix(v, "v") {
		n, err := strconv.ParseInt(strings.TrimPrefix(v, "v"), 10, 64)
		if err != nil {
			return domain.ReplayQuery{}, err
		}
		return domain.ReplayQuery{Version: &n}, nil
	}
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		return domain.ReplayQuery{Version: &n}, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return domain.ReplayQuery{}, err
	}
	return domain.ReplayQuery{At: &t}, nil
}

func parseExpectedVersion(r *http.Request) *int64 {
	v := r.Header.Get("If-Match")
	if v == "" {
		v = r.URL.Query().Get("expected_version")
	}
	if v == "" {
		return nil
	}
	v = strings.TrimPrefix(v, "v")
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func extractMetadata(r *http.Request) domain.EventMetadata {
	return domain.EventMetadata{
		Actor:          r.Header.Get("X-Actor"),
		CorrelationID:  r.Header.Get("X-Correlation-ID"),
		TraceID:        r.Header.Get("X-Trace-ID"),
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	}
}

func (s *Server) checkIdempotency(r *http.Request) (json.RawMessage, int, bool) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || s.deps.Idempotency == nil {
		return nil, 0, false
	}
	body, status, ok, err := s.deps.Idempotency.Get(r.Context(), key)
	if err != nil || !ok {
		return nil, 0, false
	}
	return body, status, true
}

func (s *Server) storeIdempotency(r *http.Request, body any, status int) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || s.deps.Idempotency == nil {
		return
	}
	_ = s.deps.Idempotency.Store(r.Context(), key, body, status, 24*time.Hour)
}

func (s *Server) handleCommandError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrVersionMismatch):
		writeError(w, http.StatusPreconditionFailed, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		s.deps.Logger.Error("command failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func (s *Server) handleQueryError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	s.deps.Logger.Error("query failed", "error", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func (s *Server) entityHistory(w http.ResponseWriter, r *http.Request, aggregateType string) {
	id := chi.URLParam(r, "id")
	events, err := s.deps.Events.History(r.Context(), aggregateType, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load history")
		return
	}
	if len(events) == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) entityTimeline(w http.ResponseWriter, r *http.Request, aggregateType string) {
	id := chi.URLParam(r, "id")
	timeline, err := s.deps.Events.Timeline(r.Context(), aggregateType, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load timeline")
		return
	}
	if len(timeline) == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, timeline)
}

func (s *Server) searchEvents(w http.ResponseWriter, r *http.Request) {
	filter := eventstore.SearchFilter{
		AggregateType: r.URL.Query().Get("aggregate_type"),
		AggregateID:   r.URL.Query().Get("aggregate_id"),
		EventType:     r.URL.Query().Get("event_type"),
		Actor:         r.URL.Query().Get("actor"),
		CorrelationID: r.URL.Query().Get("correlation_id"),
	}
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid from")
			return
		}
		filter.From = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid to")
			return
		}
		filter.To = &t
	}

	events, err := s.deps.Events.Search(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeRawJSON(w http.ResponseWriter, status int, raw json.RawMessage) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type replayRequest struct {
	At      *time.Time `json:"at"`
	Version *int64     `json:"version"`
}
