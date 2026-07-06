package grpcapi

import (
	"context"
	"net"

	temporalv1 "github.com/api-time-machine/api_time_machine/api/gen/temporal/v1"
	"github.com/api-time-machine/api_time_machine/internal/cache"
	"github.com/api-time-machine/api_time_machine/internal/domain"
	"github.com/api-time-machine/api_time_machine/internal/eventstore"
	"github.com/api-time-machine/api_time_machine/internal/order"
	"github.com/api-time-machine/api_time_machine/internal/product"
	"github.com/api-time-machine/api_time_machine/internal/projection"
	"github.com/api-time-machine/api_time_machine/internal/user"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	temporalv1.UnimplementedTemporalServiceServer
	replay    *cache.ReplayCache
	events    *eventstore.Store
	userRead  *projection.UserProjector
	prodRead  *projection.ProductProjector
	orderRead *projection.OrderProjector
}

func New(
	replay *cache.ReplayCache,
	events *eventstore.Store,
	userRead *projection.UserProjector,
	prodRead *projection.ProductProjector,
	orderRead *projection.OrderProjector,
) *Server {
	return &Server{replay: replay, events: events, userRead: userRead, prodRead: prodRead, orderRead: orderRead}
}

func (s *Server) Listen(addr string) (*grpc.Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	srv := grpc.NewServer()
	temporalv1.RegisterTemporalServiceServer(srv, s)
	go func() { _ = srv.Serve(ln) }()
	return srv, nil
}

func (s *Server) GetUser(ctx context.Context, req *temporalv1.GetEntityRequest) (*temporalv1.User, error) {
	q := entityQuery(req.At, req.Version)
	if domain.IsTemporal(q) {
		u, err := s.replay.ReplayUser(ctx, req.Id, q)
		if err != nil {
			return nil, err
		}
		return toProtoUser(u), nil
	}
	u, err := s.userRead.GetCurrent(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return toProtoUser(u), nil
}

func (s *Server) ReplayUser(ctx context.Context, req *temporalv1.ReplayRequest) (*temporalv1.User, error) {
	u, err := s.replay.ReplayUser(ctx, req.Id, replayQuery(req.At, req.Version))
	if err != nil {
		return nil, err
	}
	return toProtoUser(u), nil
}

func (s *Server) GetProduct(ctx context.Context, req *temporalv1.GetEntityRequest) (*temporalv1.Product, error) {
	q := entityQuery(req.At, req.Version)
	if domain.IsTemporal(q) {
		p, err := s.replay.Inner().ReplayProduct(ctx, req.Id, q)
		if err != nil {
			return nil, err
		}
		return toProtoProduct(p), nil
	}
	p, err := s.prodRead.GetCurrent(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return toProtoProduct(p), nil
}

func (s *Server) ReplayProduct(ctx context.Context, req *temporalv1.ReplayRequest) (*temporalv1.Product, error) {
	p, err := s.replay.Inner().ReplayProduct(ctx, req.Id, replayQuery(req.At, req.Version))
	if err != nil {
		return nil, err
	}
	return toProtoProduct(p), nil
}

func (s *Server) GetOrder(ctx context.Context, req *temporalv1.GetEntityRequest) (*temporalv1.Order, error) {
	q := entityQuery(req.At, req.Version)
	if domain.IsTemporal(q) {
		o, err := s.replay.Inner().ReplayOrder(ctx, req.Id, q)
		if err != nil {
			return nil, err
		}
		return toProtoOrder(o), nil
	}
	o, err := s.orderRead.GetCurrent(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return toProtoOrder(o), nil
}

func (s *Server) ReplayOrder(ctx context.Context, req *temporalv1.ReplayRequest) (*temporalv1.Order, error) {
	o, err := s.replay.Inner().ReplayOrder(ctx, req.Id, replayQuery(req.At, req.Version))
	if err != nil {
		return nil, err
	}
	return toProtoOrder(o), nil
}

func (s *Server) GetTimeline(ctx context.Context, req *temporalv1.TimelineRequest) (*temporalv1.TimelineResponse, error) {
	entries, err := s.events.Timeline(ctx, req.AggregateType, req.AggregateId)
	if err != nil {
		return nil, err
	}
	resp := &temporalv1.TimelineResponse{}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, &temporalv1.TimelineEntry{
			Version:   e.Version,
			EventType: e.EventType,
			EventId:   e.EventID,
			Timestamp: timestamppb.New(e.Timestamp),
			Actor:     e.Actor,
		})
	}
	return resp, nil
}

func (s *Server) SearchEvents(ctx context.Context, req *temporalv1.SearchEventsRequest) (*temporalv1.SearchEventsResponse, error) {
	events, err := s.events.Search(ctx, eventstore.SearchFilter{
		AggregateType: req.AggregateType,
		AggregateID:   req.AggregateId,
		EventType:     req.EventType,
		Actor:         req.Actor,
	})
	if err != nil {
		return nil, err
	}
	resp := &temporalv1.SearchEventsResponse{}
	for _, e := range events {
		resp.Events = append(resp.Events, &temporalv1.StoredEvent{
			Id: e.ID, AggregateType: e.AggregateType, AggregateId: e.AggregateID,
			EventType: e.EventType, Version: e.Version, Payload: e.Payload,
		})
	}
	return resp, nil
}

func entityQuery(at *timestamppb.Timestamp, version int64) domain.ReplayQuery {
	q := domain.ReplayQuery{}
	if at != nil {
		t := at.AsTime()
		q.At = &t
	}
	if version > 0 {
		q.Version = &version
	}
	return q
}

func replayQuery(at *timestamppb.Timestamp, version int64) domain.ReplayQuery {
	return entityQuery(at, version)
}

func toProtoUser(u *user.User) *temporalv1.User {
	return &temporalv1.User{Id: u.ID, Name: u.Name, Email: u.Email, Version: u.Version, Deleted: u.Deleted}
}

func toProtoProduct(p *product.Product) *temporalv1.Product {
	return &temporalv1.Product{
		Id: p.ID, Name: p.Name, Description: p.Description,
		PriceCents: p.PriceCents, Currency: p.Currency, Version: p.Version, Deleted: p.Deleted,
	}
}

func toProtoOrder(o *order.Order) *temporalv1.Order {
	items := make([]*temporalv1.OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, &temporalv1.OrderItem{
			ProductId: it.ProductID, Quantity: int32(it.Quantity), PriceCents: it.PriceCents,
		})
	}
	return &temporalv1.Order{
		Id: o.ID, UserId: o.UserID, Status: o.Status, TotalCents: o.TotalCents,
		Currency: o.Currency, Items: items, Version: o.Version, Cancelled: o.Cancelled,
	}
}
