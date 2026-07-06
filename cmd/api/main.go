package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/api-time-machine/api_time_machine/configs"
	"github.com/api-time-machine/api_time_machine/internal/api"
	"github.com/api-time-machine/api_time_machine/internal/auth"
	"github.com/api-time-machine/api_time_machine/internal/cache"
	"github.com/api-time-machine/api_time_machine/internal/diff"
	"github.com/api-time-machine/api_time_machine/internal/eventstore"
	grpcapi "github.com/api-time-machine/api_time_machine/internal/grpc"
	"github.com/api-time-machine/api_time_machine/internal/observability"
	"github.com/api-time-machine/api_time_machine/internal/order"
	"github.com/api-time-machine/api_time_machine/internal/platform"
	"github.com/api-time-machine/api_time_machine/internal/product"
	"github.com/api-time-machine/api_time_machine/internal/projection"
	"github.com/api-time-machine/api_time_machine/internal/replay"
	"github.com/api-time-machine/api_time_machine/internal/snapshot"
	"github.com/api-time-machine/api_time_machine/internal/user"
	"github.com/api-time-machine/api_time_machine/internal/worker"
	"github.com/api-time-machine/api_time_machine/pkg/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

func main() {
	cfg := configs.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	shutdownTrace, err := observability.InitTracing(ctx, cfg.OTLPEndpoint)
	if err != nil {
		logger.Error("tracing init", "error", err)
		os.Exit(1)
	}
	defer func() { _ = shutdownTrace(context.Background()) }()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if cfg.MigrateOnStart {
		migrationsDir := filepath.Join(projectRoot(), "migrations")
		if err := migrate.Up(ctx, pool, migrationsDir); err != nil {
			logger.Error("migrate", "error", err)
			os.Exit(1)
		}
		logger.Info("migrations applied")
	}

	events := eventstore.New(pool)
	snapshots := snapshot.New(pool)
	userProjector := projection.NewUserProjector(pool)
	productProjector := projection.NewProductProjector(pool)
	orderProjector := projection.NewOrderProjector(pool)

	replayEng := replay.New(events, snapshots)
	redisClient, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		logger.Warn("redis unavailable, replay cache disabled", "error", err)
	}
	if redisClient != nil {
		if err := cache.Ping(ctx, redisClient); err != nil {
			logger.Warn("redis ping failed, replay cache disabled", "error", err)
			redisClient = nil
		}
	}
	replayCache := cache.NewReplayCache(redisClient, replayEng, cfg.ReplayCacheTTL)

	userSvc := user.NewService(events, userProjector, snapshots, cfg.SnapshotEvery)
	productCmds := platform.NewCommandService(events, productProjector, snapshots, product.AggregateType, cfg.SnapshotEvery)
	orderCmds := platform.NewCommandService(events, orderProjector, snapshots, order.AggregateType, cfg.SnapshotEvery)
	productSvc := product.NewService(productCmds, productProjector)
	orderSvc := order.NewService(orderCmds, orderProjector)
	idempotency := platform.NewIdempotency(events)

	authenticator := auth.New(auth.Config{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer})

	httpServer := api.NewServer(api.Deps{
		Users: userSvc, Products: productSvc, Orders: orderSvc,
		Replay: replayCache, Diff: diff.New(), Events: events,
		UserRead: userProjector, ProductRead: productProjector, OrderRead: orderProjector,
		Idempotency: idempotency, Auth: authenticator, Logger: logger,
	})

	srv := &http.Server{
		Addr: cfg.HTTPAddr, Handler: httpServer.Router(),
		ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}

	grpcServer := grpcapi.New(replayCache, events, userProjector, productProjector, orderProjector)
	grpcSrv, err := grpcServer.Listen(cfg.GRPCAddr)
	if err != nil {
		logger.Error("grpc listen", "error", err)
		os.Exit(1)
	}

	var nc *nats.Conn
	if cfg.NATSURL != "" {
		nc, err = nats.Connect(cfg.NATSURL)
		if err != nil {
			logger.Warn("nats unavailable", "error", err)
		}
	}

	if cfg.WorkersEnabled {
		outbox := worker.NewOutboxPublisher(pool, nc, logger, cfg.NATSTopic)
		snapWorker := worker.NewSnapshotBuilder(events, snapshots, replayEng, logger)
		cacheWarmer := worker.NewCacheWarmer(events, replayEng, redisClient, cfg.ReplayCacheTTL, logger)
		go outbox.Run(ctx)
		go snapWorker.Run(ctx)
		go cacheWarmer.Run(ctx)
	}

	go func() {
		logger.Info("http server started", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server", "error", err)
			os.Exit(1)
		}
	}()
	logger.Info("grpc server started", "addr", cfg.GRPCAddr)

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	grpcSrv.GracefulStop()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "error", err)
	}
	if nc != nil {
		nc.Close()
	}
	if redisClient != nil {
		_ = redisClient.Close()
	}
	logger.Info("stopped")
}

func projectRoot() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}
