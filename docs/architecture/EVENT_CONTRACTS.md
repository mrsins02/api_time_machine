# Event Contracts

## Naming Convention

`{Aggregate}{PastTenseVerb}` — e.g. `UserCreated`, `PriceChanged`

## User Events

| Event | Payload |
|-------|---------|
| UserCreated | `{ name, email }` |
| UserUpdated | `{ name?, email?, address?, avatar? }` |
| UserDeleted | `{}` |
| EmailChanged | `{ email }` |
| AddressChanged | `{ address }` |
| AvatarChanged | `{ avatar }` |
| PasswordChanged | `{}` (hash never in read model) |

## Product Events

| Event | Payload |
|-------|---------|
| ProductCreated | `{ name, description, price_cents, currency }` |
| ProductUpdated | `{ name?, description?, price_cents?, currency? }` |
| ProductDeleted | `{}` |
| PriceChanged | `{ price_cents, currency? }` |

## Order Events

| Event | Payload |
|-------|---------|
| OrderCreated | `{ user_id, currency, total_cents, items[] }` |
| OrderUpdated | `{ status? }` |
| OrderCancelled | `{}` |
| OrderItemAdded | `{ item: { product_id, quantity, price_cents } }` |
| OrderItemRemoved | `{ product_id }` |

## Metadata (all events)

```json
{
  "actor": "user@example.com",
  "correlation_id": "req-uuid",
  "trace_id": "otel-trace-id",
  "idempotency_key": "key-uuid"
}
```

## NATS Message Format

Published by outbox worker to topic `atm.events` (configurable):

```json
{
  "event_id": 1042,
  "aggregate_type": "user",
  "aggregate_id": "42",
  "event_type": "EmailChanged",
  "version": 3,
  "payload": { "email": "new@example.com" }
}
```

## Versioning Policy

- New event types may be added without breaking replay (unknown events ignored)
- Existing event payload shapes must remain backward compatible
- Never rename or remove published event types

## Implementation

- Domain constants: aggregate event type strings in `internal/{user,product,order}/`
- Cross-service constants: `pkg/events/contracts.go`
