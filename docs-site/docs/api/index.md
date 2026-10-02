# Politburo API

The Politburo API is the machine contract for health checks, live game caches, user
registration, signed links, and operator actions exposed in the OpenAPI description.

## Interactive reference

**[Open API reference (ReDoc)](reference/index.html)** — browse paths, schemas, and server URLs
in the browser. The renderer and spec are bundled with this site (no external CDN required).

## Download

- [openapi.yaml](reference/openapi.yaml) — same document Politburo codegen uses after bundling
  from `openapi/politburo/` in the repository.

## Authentication (summary)

| Mechanism | Used for |
|-----------|----------|
| `X-API-Key` | Bot and other trusted clients on `/api/v1/*` |
| `X-Discord-User-Id`, `X-Discord-Server-Id` | Discord context on bot-originated requests |
| `session_id` cookie | Browser sessions after signed-link login (selected routes) |

Public health routes under `/health/*` do not require auth. See the reference for per-operation
security.
