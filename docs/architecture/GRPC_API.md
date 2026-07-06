# gRPC API

Service: `temporal.v1.TemporalService`  
Port: `:9090` (default)

## Proto Definition

`api/proto/temporal/v1/temporal.proto`

Generate stubs:
```bash
make proto
```

## RPCs

| RPC | Request | Response |
|-----|---------|----------|
| GetUser | GetEntityRequest | User |
| ReplayUser | ReplayRequest | User |
| GetProduct | GetEntityRequest | Product |
| ReplayProduct | ReplayRequest | Product |
| GetOrder | GetEntityRequest | Order |
| ReplayOrder | ReplayRequest | Order |
| GetTimeline | TimelineRequest | TimelineResponse |
| SearchEvents | SearchEventsRequest | SearchEventsResponse |

## Temporal Queries

`GetEntityRequest` supports optional temporal fields:

```protobuf
message GetEntityRequest {
  string id = 1;
  google.protobuf.Timestamp at = 2;  // optional
  int64 version = 3;                 // optional
}
```

If `at` or `version` is set, the server replays rather than reading the projection.

## Example (grpcurl)

```bash
grpcurl -plaintext -d '{"id":"42","version":3}' \
  localhost:9090 temporal.v1.TemporalService/GetUser
```

## Implementation

- Generated: `api/gen/temporal/v1/`
- Server: `internal/grpc/server.go`
