# Infinite Experiment — Architecture Reference

Use this skill when you need workspace context before implementing or reviewing anything. It covers the three repos, prod deployment, and the patterns every engineer must follow.

---

## Workspace layout

| Directory | Purpose | Language |
|---|---|---|
| `politburo/` | Backend API + Vizburo web UI | Go |
| `comrade-bot/` | Discord bot | TypeScript |
| `labour-bureau/` | Infrastructure | Docker Compose / Podman |

Key planning docs: `politburo/CLAUDE.md`, `politburo/RELEASE_V1.0_PLAN.md`.

---

## Production deployment (labour-bureau/prod/)

**Container runtime**: Podman (not Docker). Systemd services (`labour-bureau-compose.service`, `labour-bureau-services.service`) manage lifecycle with `--restart unless-stopped`.

**Network**: All services share an `internal` bridge network. Caddy is the only public entry point (ports 80/443 on host).

**Domains** (via Caddy + Let's Encrypt):
- `comradebot.cc` → politburo (`:8080`) — proxies `/api/*`, `/public/*`, `/auth/*`, `/dashboard/*`, `/static/*`, `/ui/api/*`
- `jobs.comradebot.cc` → vizburo/jobs service (`:3001`)
- `monitor.comradebot.cc` → Grafana (`:3000`)

**Security**: `/metrics` endpoint is blocked at Caddy for `comradebot.cc` — Prometheus scrapes internally only.

**Retention**: Prometheus 30d, Loki 14d. Dev: Prometheus 7d, Loki 7d.

**Prod vs dev**:
- Prod: `docker-compose.prod.yml` (PostgreSQL `healthcheck` required before politburo starts, Redis with auth password, `restart: unless-stopped` on all services).
- Dev: `docker-compose.dev.yml` (hot reload via Air, pgAdmin on `:5050`, no resource limits, politburo may run natively via tmux instead of in container).

---

## Politburo (Go backend)

### Key packages

| Package | Path | Purpose |
|---|---|---|
| Router | `internal/routes/` | Chi router setup, middleware wiring |
| API handlers | `internal/api/` | HTTP handler structs (feature-scoped) |
| Generated handlers | `internal/api/generated/<domain>/` | oapi-codegen output — never hand-edit |
| Services | `internal/services/` | Business logic |
| Repositories | `internal/db/repositories/` | GORM data access |
| Workers | `internal/workers/` | Background goroutines |
| Jobs | `internal/jobs/` | Scheduled cron jobs |
| Constants | `internal/constants/` | All typed string/int constants |
| Common utils | `internal/common/` | `RespondSuccess`, `RespondError`, cache helpers |
| Infra/logging | `infra/logging/` | Zap-backed structured logger (already live) |
| Infra/metrics | `infra/metrics/` | Prometheus `MetricsRegistry` (already live) |
| Infra (other) | `infra/{cache,db,redis,queue,scheduler,security,session,templates}/` | Shared infrastructure primitives |
| OpenAPI specs | `api/openapi/` | `<domain>.yaml` specs (source of truth for new REST endpoints) |

### Handler pattern

Feature-scoped structs, injected at `RegisterRoutes` — no globals:
```go
type PirepHandlers struct {
    svc  *services.PirepService
    repo *repositories.VAGormRepository
}
```

### Response helpers (always use these)

```go
common.RespondSuccess(w, initTime, "message", data)
common.RespondError(w, initTime, err, "message", http.StatusBadRequest)
common.RespondErrorWithData(w, initTime, err, "message", partialData)
common.RespondPermissionDenied(w, "admin")
```
Never use raw `http.Error`.

### Auth

Claims attached to context via `auth.SetUserClaims`, retrieved via `auth.GetUserClaims`. Auth uses API keys: `X-API-Key`, `X-Server-Id`, `X-Discord-Id` headers. Never read these headers directly in handlers.

Role hierarchy (Postgres ENUM): `pilot` → `staff` → `admin`. Defined in `internal/constants/roles.go`.

### Database

GORM only (`db.PgDB`). No sqlx. Repositories in `internal/db/repositories/`. Migrations in `internal/db/migrations/` — applied sequentially by number.

### Caching

- `RedisCacheService` for Redis-backed cache (TTL-managed).
- `CacheService` for in-memory (`go-cache`).
- Cache key patterns are typed constants in `internal/constants/` — no bare strings.

### Logging

Use `infra/logging` — backed by `go.uber.org/zap` (already in `go.mod`):
```go
logging.Info("message", "key", value)
logging.Warn("message", "key", value)
logging.Error("message", "key", value)
logging.WithRequest(requestID, serverID, userID, endpoint).Info("message")
```
Levels: INFO for lifecycle, WARN for recoverable errors, ERROR for failures. Never log secrets, auth headers, or full request bodies in production code paths.

### Metrics

`infra/metrics.MetricsRegistry` is already live. Add new metrics to the registry — do not create a second registry. Existing metric families:
- HTTP: `politburo_http_requests_total`, `politburo_http_request_duration_seconds`, `politburo_http_requests_in_flight`
- DB: `politburo_db_queries_total`, `politburo_db_query_duration_seconds`
- Cache: `politburo_cache_hits_total`, `politburo_cache_misses_total`
- Queue: `politburo_queue_depth` (labeled by `queue_name`, `queue_type`), plus enqueue/dequeue/error counters
- Sync jobs: `politburo_sync_job_records_processed`, `politburo_sync_job_duration_seconds`

### Workers

Must accept `context.Context` for cancellation. Use buffered channels. Log queue overflow. Use `WorkerManager` pattern for graceful shutdown on SIGTERM/SIGINT.

### New REST endpoints — use oapi-codegen

All new JSON-returning REST endpoints are spec-driven. See `/oapi-codegen` skill for the full workflow. HTML-emitting Vizburo/dashboard template handlers keep the existing `http.HandlerFunc` pattern.

### Constants rule

Every cache key, route string, error message, and config key is a typed constant in `internal/constants/`. No bare string literals in business logic.

---

## Comrade-Bot (TypeScript Discord bot)

### Architecture

```
src/
  index.ts                  — entry point, graceful SIGINT/SIGTERM
  bot/BotClient.ts          — Discord client init, delegates to InteractionRouter
  handlers/InteractionRouter.ts — routes all interactions (commands, modals, buttons, select menus)
  commands/<name>.ts        — one file per slash command (exports data + execute)
  commands/<name>Handler.ts — modal/button handlers triggered by a command flow
  services/apiService.ts    — all HTTP calls to Politburo (single gateway)
  types/                    — response types, DiscordInteraction wrapper
  configs/constants.ts      — CUSTOM_IDS and other string constants
  configs/commandMap.ts     — maps command names to handler objects
  helpers/                  — UnauthorizedError, PermissionDeniedError, NotFoundError, utils
```

### Key patterns

- **InteractionRouter** dispatches all incoming Discord interactions (commands, modals, buttons, select menus) by type and `customId`.
- **DiscordInteraction** is a wrapper around `discord.js Interaction` — always pass this to handlers, never the raw interaction.
- **apiService.ts** is the only place that makes HTTP calls to Politburo. Commands call `ApiService.someMethod()`, not `fetch()` directly.
- **Error handling**: Three custom exception classes (`UnauthorizedError`, `PermissionDeniedError`, `NotFoundError`) — throw these from `apiService.ts` so commands can handle them with user-friendly Discord messages.
- **CUSTOM_IDS**: All button/modal custom ID strings live in `configs/constants.ts`. Parse with `split("_")` by convention; prefix determines routing.
- **Command registration**: Separate `deploy-commands.ts` script — run after adding new slash commands.
- **Logging**: `console.log` for lifecycle, `console.error` for errors, `console.warn` for warnings. Do not add a logging library without discussion.

### Adding a new command

1. `src/commands/<name>.ts` — export `data` (SlashCommandBuilder) + `execute(interaction: DiscordInteraction)`
2. `src/commands/<name>ModalHandler.ts` / `<name>ButtonHandler.ts` — if the command opens modals or buttons
3. Register in `src/configs/commandMap.ts`
4. Register modal/button handlers in `InteractionRouter.ts`
5. Any new API calls go in `src/services/apiService.ts` with typed response types in `src/types/Responses.ts`
6. Run deploy script to register slash commands with Discord

---

## Infrastructure conventions (labour-bureau)

- **Docker labels**: Every container in dev compose carries `labels: { service: "<name>", env: "dev" }` — Promtail uses these for log routing. Prod containers follow the same convention.
- **Health checks**: All stateful services (db, redis) must have `healthcheck` blocks in compose. Politburo depends on db/redis health before starting.
- **Secrets**: Real secrets go in `.env.*` files (gitignored). Document all required vars in `.env.example` / `.env.prod.example`. Never commit credentials.
- **New service checklist**: HEALTHCHECK in Dockerfile, `healthcheck` in compose, Docker labels, `restart: unless-stopped` in prod, entry in Caddy if publicly reachable, Prometheus scrape target in `prometheus.prod.yml` if it exports metrics.
