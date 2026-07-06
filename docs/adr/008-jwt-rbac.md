# ADR-008: JWT + RBAC for API Authorization

**Status:** Accepted  
**Date:** 2026-04-18

## Context

Temporal data is sensitive. Replay and history endpoints expose historical state that may require authorization beyond simple API keys.

## Decision

Implement **JWT authentication** with **role-based access control**:

| Role | Permissions |
|------|-------------|
| `admin` | All operations |
| `writer` | Create, update, delete commands |
| `reader` | Read, replay, diff, history, timeline |

Auth is **optional** — when `JWT_SECRET` is empty, all endpoints are open (development mode).

Token issuance: `POST /auth/token { "subject", "role" }`

## Consequences

**Positive**
- Stateless auth scales horizontally
- Role separation protects destructive commands vs reads
- Dev-friendly when auth disabled

**Negative**
- No built-in token revocation (short TTL recommended)
- gRPC auth not enforced in v1 (REST only)
- `POST /auth/token` is open when auth enabled (dev convenience — should be restricted in production)

## Public Endpoints

`/health`, `/metrics`, `/ui/*` — always unauthenticated.

## Implementation

`internal/auth/jwt.go`
