package observability

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "atm_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})

	ReplayDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "atm_replay_duration_seconds",
		Help:    "Replay operation duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"aggregate_type"})

	ProjectionLag = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "atm_projection_lag_events",
		Help: "Unpublished outbox events",
	})

	SnapshotLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "atm_snapshot_latency_seconds",
		Help:    "Snapshot build latency",
		Buckets: prometheus.DefBuckets,
	})
)

func InitTracing(ctx context.Context, endpoint string) (func(context.Context) error, error) {
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("api-time-machine"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		HTTPRequests.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(ww.Status())).Inc()
		_ = start
	})
}

func LoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
