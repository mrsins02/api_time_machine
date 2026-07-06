.PHONY: build test run up down proto bench

build:
	go build -o bin/api ./cmd/api

test:
	go test ./... -count=1

bench:
	go test ./... -bench=. -benchmem -run=^$

run: build
	DATABASE_URL="postgres://atm:atm@localhost:5432/atm?sslmode=disable" \
	REDIS_URL="redis://localhost:6379/0" \
	NATS_URL="nats://localhost:4222" \
	./bin/api

up:
	docker compose -f deploy/docker-compose.yml up -d --build

down:
	docker compose -f deploy/docker-compose.yml down

proto:
	protoc -I api/proto --go_out=api/gen --go_opt=paths=source_relative \
		--go-grpc_out=api/gen --go-grpc_opt=paths=source_relative \
		api/proto/temporal/v1/temporal.proto
