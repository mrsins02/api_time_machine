package configs

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	GRPCAddr        string
	DatabaseURL     string
	RedisURL        string
	NATSURL         string
	NATSTopic       string
	JWTSecret       string
	JWTIssuer       string
	OTLPEndpoint    string
	SnapshotEvery   int
	ReplayCacheTTL  time.Duration
	MigrateOnStart  bool
	WorkersEnabled  bool
}

func Load() Config {
	return Config{
		HTTPAddr:       env("HTTP_ADDR", ":8080"),
		GRPCAddr:       env("GRPC_ADDR", ":9090"),
		DatabaseURL:    env("DATABASE_URL", "postgres://atm:atm@localhost:5432/atm?sslmode=disable"),
		RedisURL:       env("REDIS_URL", "redis://localhost:6379/0"),
		NATSURL:        env("NATS_URL", "nats://localhost:4222"),
		NATSTopic:      env("NATS_TOPIC", "atm.events"),
		JWTSecret:      env("JWT_SECRET", ""),
		JWTIssuer:      env("JWT_ISSUER", "api-time-machine"),
		OTLPEndpoint:   env("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		SnapshotEvery:  envInt("SNAPSHOT_EVERY", 50),
		ReplayCacheTTL: envDuration("REPLAY_CACHE_TTL", 5*time.Minute),
		MigrateOnStart: envBool("MIGRATE_ON_START", true),
		WorkersEnabled: envBool("WORKERS_ENABLED", true),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
