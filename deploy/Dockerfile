FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /bin/api /app/api
COPY migrations /app/migrations

ENV HTTP_ADDR=:8080
ENV GRPC_ADDR=:9090
EXPOSE 8080 9090
CMD ["/app/api"]
