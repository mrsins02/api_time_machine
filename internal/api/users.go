package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/observability"
	"github.com/api-time-machine/api_time_machine/internal/user"
	"github.com/go-chi/chi/v5"
)

type createUserRequest struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type updateUserRequest struct {
	Name    *string `json:"name"`
	Email   *string `json:"email"`
	Address *string `json:"address"`
	Avatar  *string `json:"avatar"`
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ID == "" || req.Name == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "id, name, and email are required")
		return
	}
	if body, status, ok := s.checkIdempotency(r); ok {
		writeRawJSON(w, status, body)
		return
	}
	u, err := s.deps.Users.Create(r.Context(), user.CreateCommand{
		ID: req.ID, Name: req.Name, Email: req.Email, Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	s.storeIdempotency(r, u, http.StatusCreated)
	writeJSON(w, http.StatusCreated, u)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body, status, ok := s.checkIdempotency(r); ok {
		writeRawJSON(w, status, body)
		return
	}
	u, err := s.deps.Users.Update(r.Context(), user.UpdateCommand{
		ID: id, Name: req.Name, Email: req.Email, Address: req.Address, Avatar: req.Avatar,
		ExpectedVersion: parseExpectedVersion(r), Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	s.storeIdempotency(r, u, http.StatusOK)
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if body, status, ok := s.checkIdempotency(r); ok {
		writeRawJSON(w, status, body)
		return
	}
	u, err := s.deps.Users.Delete(r.Context(), user.DeleteCommand{
		ID: id, ExpectedVersion: parseExpectedVersion(r), Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	s.storeIdempotency(r, u, http.StatusOK)
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	query, err := parseReplayQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if query.isTemporal() {
		start := time.Now()
		u, err := s.deps.Replay.ReplayUser(r.Context(), id, query.toDomain())
		observability.ReplayDuration.WithLabelValues("user").Observe(time.Since(start).Seconds())
		if err != nil {
			s.handleQueryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, u)
		return
	}
	u, err := s.deps.UserRead.GetCurrent(r.Context(), id)
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) userHistory(w http.ResponseWriter, r *http.Request) {
	s.entityHistory(w, r, user.AggregateType)
}

func (s *Server) userTimeline(w http.ResponseWriter, r *http.Request) {
	s.entityTimeline(w, r, user.AggregateType)
}

func (s *Server) userDiff(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fromQ, err := parseTimeOrVersionParam(r, "from")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from parameter")
		return
	}
	toQ, err := parseTimeOrVersionParam(r, "to")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to parameter")
		return
	}
	fromUser, err := s.deps.Replay.ReplayUser(r.Context(), id, fromQ)
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	toUser, err := s.deps.Replay.ReplayUser(r.Context(), id, toQ)
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.deps.Diff.Compare(fromUser.ToMap(), toUser.ToMap()))
}

func (s *Server) userReplay(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req replayRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	u, err := s.deps.Replay.ReplayUser(r.Context(), id, domain.ReplayQuery{At: req.At, Version: req.Version})
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) userPreview(w http.ResponseWriter, r *http.Request) {
	s.userReplay(w, r)
}
