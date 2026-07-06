package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/auth"
	"github.com/api-time-machine/api_time_machine/internal/cache"
	"github.com/api-time-machine/api_time_machine/internal/diff"
	"github.com/api-time-machine/api_time_machine/internal/eventstore"
	"github.com/api-time-machine/api_time_machine/internal/observability"
	"github.com/api-time-machine/api_time_machine/internal/order"
	"github.com/api-time-machine/api_time_machine/internal/platform"
	"github.com/api-time-machine/api_time_machine/internal/product"
	"github.com/api-time-machine/api_time_machine/internal/projection"
	"github.com/api-time-machine/api_time_machine/internal/user"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Deps struct {
	Users       *user.Service
	Products    *product.Service
	Orders      *order.Service
	Replay      *cache.ReplayCache
	Diff        *diff.Engine
	Events      *eventstore.Store
	UserRead    *projection.UserProjector
	ProductRead *projection.ProductProjector
	OrderRead   *projection.OrderProjector
	Idempotency *platform.Idempotency
	Auth        *auth.Authenticator
	Logger      *slog.Logger
}

type Server struct {
	deps Deps
}

func NewServer(deps Deps) *Server {
	return &Server{deps: deps}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(rateLimitMiddleware(300, time.Minute))
	r.Use(observability.HTTPMetricsMiddleware)
	r.Use(observability.LoggerMiddleware(s.deps.Logger))

	if s.deps.Auth != nil {
		r.Use(s.deps.Auth.Middleware)
	}

	r.Get("/health", s.health)
	r.Handle("/metrics", observability.MetricsHandler())
	r.Handle("/ui/*", http.StripPrefix("/ui", uiHandler()))

	r.Route("/auth", func(r chi.Router) {
		r.Post("/token", s.issueToken)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireWrite)
		s.mountUserCommands(r)
		s.mountProductCommands(r)
		s.mountOrderCommands(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireReplay)
		s.mountUserQueries(r)
		s.mountProductQueries(r)
		s.mountOrderQueries(r)
		r.Get("/events", s.searchEvents)
	})

	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) issueToken(w http.ResponseWriter, r *http.Request) {
	if s.deps.Auth == nil || !s.deps.Auth.Enabled() {
		writeError(w, http.StatusNotImplemented, "auth disabled")
		return
	}
	var req struct {
		Subject string `json:"subject"`
		Role    string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	role := auth.Role(req.Role)
	if role == "" {
		role = auth.RoleReader
	}
	token, err := s.deps.Auth.Issue(req.Subject, role, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *Server) mountUserCommands(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/", s.createUser)
		r.Put("/{id}", s.updateUser)
		r.Delete("/{id}", s.deleteUser)
	})
}

func (s *Server) mountUserQueries(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Get("/{id}", s.getUser)
		r.Get("/{id}/history", s.userHistory)
		r.Get("/{id}/timeline", s.userTimeline)
		r.Get("/{id}/diff", s.userDiff)
		r.Post("/{id}/replay", s.userReplay)
		r.Post("/{id}/preview", s.userPreview)
	})
}

func (s *Server) mountProductCommands(r chi.Router) {
	r.Route("/products", func(r chi.Router) {
		r.Post("/", s.createProduct)
		r.Put("/{id}", s.updateProduct)
		r.Delete("/{id}", s.deleteProduct)
	})
}

func (s *Server) mountProductQueries(r chi.Router) {
	r.Route("/products", func(r chi.Router) {
		r.Get("/{id}", s.getProduct)
		r.Get("/{id}/history", s.productHistory)
		r.Get("/{id}/timeline", s.productTimeline)
		r.Get("/{id}/diff", s.productDiff)
		r.Post("/{id}/replay", s.productReplay)
	})
}

func (s *Server) mountOrderCommands(r chi.Router) {
	r.Route("/orders", func(r chi.Router) {
		r.Post("/", s.createOrder)
		r.Put("/{id}", s.updateOrder)
		r.Post("/{id}/cancel", s.cancelOrder)
		r.Post("/{id}/items", s.addOrderItem)
	})
}

func (s *Server) mountOrderQueries(r chi.Router) {
	r.Route("/orders", func(r chi.Router) {
		r.Get("/{id}", s.getOrder)
		r.Get("/{id}/history", s.orderHistory)
		r.Get("/{id}/timeline", s.orderTimeline)
		r.Get("/{id}/diff", s.orderDiff)
		r.Post("/{id}/replay", s.orderReplay)
	})
}
