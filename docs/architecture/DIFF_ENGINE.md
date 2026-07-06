# Diff Engine

## Purpose

Compare two aggregate states and report which fields changed — enabling "what changed between Monday and Tuesday?" queries.

## API

```
GET /users/42/diff?from=2026-04-18T10:00:00Z&to=2026-04-18T15:00:00Z
GET /users/42/diff?from=1&to=5
GET /users/42/diff?from=v1&to=v5
```

`from` and `to` accept RFC3339 timestamps or version numbers.

## Response

```json
{
  "changed": ["email", "name"],
  "from": { "id": "42", "email": "old@mail.com", "name": "Ali" },
  "to":   { "id": "42", "email": "ali@gmail.com", "name": "Ali B" },
  "fields": {
    "email": { "from": "old@mail.com", "to": "ali@gmail.com" },
    "name":  { "from": "Ali", "to": "Ali B" }
  }
}
```

## Algorithm

1. Replay aggregate at `from` query → `fromMap`
2. Replay aggregate at `to` query → `toMap`
3. Deep-compare all keys using `reflect.DeepEqual`
4. Report added, removed, and modified fields

## UI Integration

The embedded UI (`/ui/`) shows inline diff during replay animation:
- Green `+` for added fields
- Red `-` for removed fields
- Yellow `~` for changed fields

## Implementation

`internal/diff/engine.go`
