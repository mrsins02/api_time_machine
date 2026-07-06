package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/observability"
	"github.com/api-time-machine/api_time_machine/internal/order"
	"github.com/api-time-machine/api_time_machine/internal/product"
	"github.com/go-chi/chi/v5"
)

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		PriceCents  int64  `json:"price_cents"`
		Currency    string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	p, err := s.deps.Products.Create(r.Context(), product.CreateCommand{
		ID: req.ID, Name: req.Name, Description: req.Description,
		PriceCents: req.PriceCents, Currency: req.Currency, Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) updateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		PriceCents  *int64  `json:"price_cents"`
		Currency    *string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	p, err := s.deps.Products.Update(r.Context(), product.UpdateCommand{
		ID: id, Name: req.Name, Description: req.Description,
		PriceCents: req.PriceCents, Currency: req.Currency,
		ExpectedVersion: parseExpectedVersion(r), Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.deps.Products.Delete(r.Context(), product.DeleteCommand{
		ID: id, ExpectedVersion: parseExpectedVersion(r), Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	query, err := parseReplayQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if query.isTemporal() {
		start := time.Now()
		p, err := s.deps.Replay.Inner().ReplayProduct(r.Context(), id, query.toDomain())
		observability.ReplayDuration.WithLabelValues("product").Observe(time.Since(start).Seconds())
		if err != nil {
			s.handleQueryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
		return
	}
	p, err := s.deps.ProductRead.GetCurrent(r.Context(), id)
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) productHistory(w http.ResponseWriter, r *http.Request) {
	s.entityHistory(w, r, product.AggregateType)
}

func (s *Server) productTimeline(w http.ResponseWriter, r *http.Request) {
	s.entityTimeline(w, r, product.AggregateType)
}

func (s *Server) productDiff(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fromQ, _ := parseTimeOrVersionParam(r, "from")
	toQ, _ := parseTimeOrVersionParam(r, "to")
	from, err := s.deps.Replay.Inner().ReplayProduct(r.Context(), id, fromQ)
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	to, err := s.deps.Replay.Inner().ReplayProduct(r.Context(), id, toQ)
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.deps.Diff.Compare(from.ToMap(), to.ToMap()))
}

func (s *Server) productReplay(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req replayRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	p, err := s.deps.Replay.Inner().ReplayProduct(r.Context(), id, domain.ReplayQuery{At: req.At, Version: req.Version})
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string       `json:"id"`
		UserID   string       `json:"user_id"`
		Currency string       `json:"currency"`
		Items    []order.Item `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	o, err := s.deps.Orders.Create(r.Context(), order.CreateCommand{
		ID: req.ID, UserID: req.UserID, Items: req.Items, Currency: req.Currency, Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) updateOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Status *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	o, err := s.deps.Orders.Update(r.Context(), order.UpdateCommand{
		ID: id, Status: req.Status, ExpectedVersion: parseExpectedVersion(r), Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := s.deps.Orders.Cancel(r.Context(), order.CancelCommand{
		ID: id, ExpectedVersion: parseExpectedVersion(r), Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) addOrderItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Item order.Item `json:"item"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	o, err := s.deps.Orders.AddItem(r.Context(), order.AddItemCommand{
		ID: id, Item: req.Item, ExpectedVersion: parseExpectedVersion(r), Metadata: extractMetadata(r),
	})
	if err != nil {
		s.handleCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	query, err := parseReplayQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if query.isTemporal() {
		start := time.Now()
		o, err := s.deps.Replay.Inner().ReplayOrder(r.Context(), id, query.toDomain())
		observability.ReplayDuration.WithLabelValues("order").Observe(time.Since(start).Seconds())
		if err != nil {
			s.handleQueryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, o)
		return
	}
	o, err := s.deps.OrderRead.GetCurrent(r.Context(), id)
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) orderHistory(w http.ResponseWriter, r *http.Request) {
	s.entityHistory(w, r, order.AggregateType)
}

func (s *Server) orderTimeline(w http.ResponseWriter, r *http.Request) {
	s.entityTimeline(w, r, order.AggregateType)
}

func (s *Server) orderDiff(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fromQ, _ := parseTimeOrVersionParam(r, "from")
	toQ, _ := parseTimeOrVersionParam(r, "to")
	from, err := s.deps.Replay.Inner().ReplayOrder(r.Context(), id, fromQ)
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	to, err := s.deps.Replay.Inner().ReplayOrder(r.Context(), id, toQ)
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.deps.Diff.Compare(from.ToMap(), to.ToMap()))
}

func (s *Server) orderReplay(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req replayRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	o, err := s.deps.Replay.Inner().ReplayOrder(r.Context(), id, domain.ReplayQuery{At: req.At, Version: req.Version})
	if err != nil {
		s.handleQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}
