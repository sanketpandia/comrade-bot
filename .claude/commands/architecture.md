# Comrade Bot monorepo — Architecture Reference

Use this before planning, implementing, or reviewing anything. It describes the **current** monorepo (`comrade-bot` on GitHub), how its parts depend on each other, and known traps. Source code wins when this file disagrees; fix this file when you notice drift.

> The old three-repo layout (`politburo/`, `comrade-bot/`, `labour-bureau/` as siblings), `politburo/CLAUDE.md`, `RELEASE_V1.0_PLAN.md`, `internal/routes/`, `internal/platform/`, `infra/liveapi`, Vizburo, GORM, Zap, Watermill, and `api/openapi/<domain>.yaml` **no longer exist**. Many files under `docs/` (notably `docs/standards.md`, `docs/politburo/**`, `docs/infra/**`) still describe them — verify against code before trusting them.

---

## Layout

| Path | Role |
|---|---|
| `openapi/politburo/` | **Source** of the Politburo public API contract (split files) |
| `openapi/politburo.yaml` | Committed Redocly **bundle** of the above — generated, never hand-edit |
| `openapi/infinite-flight/openapi.yaml` | Upstream Infinite Flight Live API spec (client generation only) |
| `cicd/Makefile` | All codegen/build targets; root `Makefile` just forwards to it |
| `cicd/package.json` | Pinned Redocly CLI (bundler) |
| `cicd/tools/go.mod` | Pinned Go tools: `oapi-codegen`, `air` (run via `go tool`) |
| `cicd/codegen/*.yaml` | oapi-codegen configs (Politburo server, IF client) |
| `services/politburo/` | Go module `infinite-experiment/politburo`: API, jobs, embedded SSR UI |
| `services/comrade-bot-discord/` | TypeScript Discord bot |
| `infra/dev/` | Local Compose backing services, Prometheus/Loki/Promtail/Grafana config, `start-dev.sh` |
| `infra/prod/` | Podman Compose, Caddy, systemd units, deploy scripts, env templates, prod Grafana |
| `docs/` | Dev notes (`docs/development/openapi.md` is current; much else is stale) |
| `.github/workflows/` | `openapi.yml`, `politburo.yml`, `discord-bot.yml` |
| `_monorepo-backup/` | Bare git backups of the three old repos — never edit, never import from |

---

## Dependency flow

### Contract → code

```
openapi/politburo/**            (edit here)
   │  make openapi-bundle        (Redocly, cicd/package.json)
   ▼
openapi/politburo.yaml          (COMMITTED bundle; CI fails if stale)
   ├─ make generate-politburo   (oapi-codegen, cicd/codegen/politburo-oapi-codegen.yaml)
   │    ▼ services/politburo/internal/api/generated/politburo/server.gen.go   (gitignored)
   └─ npm run api:generate      (openapi-typescript, bot package.json)
        ▼ services/comrade-bot-discord/src/generated/politburo-api.ts         (gitignored)

openapi/infinite-flight/openapi.yaml
   │  make generate-infinite-flight
   ▼ services/politburo/internal/api/generated/infiniteflight/client.gen.go   (gitignored)
```

- `make generate` = bundle, then all three generators. `make check` = generate + Politburo tests. `make openapi-bundle-check` = fail if bundle is stale.
- Generated Go/TS is **gitignored**: a fresh checkout does not compile until `make generate` runs.
- Politburo codegen config is `models` + `chi-server` (plain `ServerInterface`, **not** strict-server).

### Build & CI

- Docker builds use the **repo root** as context (`docker build -f services/<svc>/Dockerfile .`).
- Politburo `Dockerfile`/`Dockerfile.dev` copy `Makefile`, `cicd/{Makefile,codegen,tools}`, `openapi/` and run `make -C cicd generate-politburo generate-infinite-flight`. They have **no Node**, so they read the committed bundle — never make those targets depend on `openapi-bundle`.
- Bot `Dockerfile` copies `openapi/` and runs `npm run api:generate`.
- CI path filters include `openapi/**`, `cicd/**`, `Makefile`; changing any of them triggers Politburo, bot, and OpenAPI workflows.

### Runtime

```
Discord ─► comrade-bot-discord ──HTTP (API_URL, X-API-Key, X-Discord-User-Id, X-Discord-Server-Id)──► Politburo
                  │ :9091 /metrics, /healthz                                                          │ :8082 dev / :8080 prod
                  ▼                                                                                   ├─► Postgres (database/sql + pgx; DB politburo_next)
             Prometheus ◄───────────────────────── scrape /metrics ────────────────────────────────── ├─► Redis (cache, API-key status, sessions, tickets, rate limit)
                  ▼                                                                                   └─► Infinite Flight Live API (jobs only)
               Grafana ◄── Loki ◄── Promtail (dev: /tmp/politburo.log + journal; prod: journal)
```

---

## Politburo (`services/politburo`)

Boot: `cmd/politburo/main.go` → `config.Load()` → `app.New()` (composition root) → `transport/http.NewServer(app).Run()`.

| Package | Purpose |
|---|---|
| `internal/config` | All env vars; `*_FILE` secret variants; defaults (`PORT=8082`, `PG_DB=politburo_next`, `JOBS_ENABLED=false`) |
| `internal/app` | **Only** place dependencies are constructed and wired (`App` struct) |
| `internal/transport/http/server.go` | Router, global middleware, generated-interface adapter `apiHandler`, hand-mounted route groups |
| `internal/transport/http/api/<feature>` | JSON handlers (`gamesessions`, `gameflights`, `health`, `signedlink`, `identity`) |
| `internal/transport/http/api/cachedresponse` | `{data:{availableFilters,result,history?,meta,pagination?}}` shape |
| `internal/transport/http/response` | `WriteJSON`, `WriteError` → `{error:{code,message}}` |
| `internal/transport/http/middleware` | Access log+metrics, CORS, `AuthenticateAPI`, Discord context, membership enrichment, roles, operator gate, rate limit |
| `internal/transport/http/ui` + `internal/ui` | SSR handlers; embedded templates/static (`go:embed`), `ui.Renderer` |
| `internal/access/auth` | `Claims` in context, signed-link tickets |
| `internal/access/session` | Redis browser sessions, `session_id` cookie |
| `internal/access/apikeys` | DB + Redis API key lookup |
| `internal/livegame/{flights,sessions,liveries}` | Cache-backed IF live game domain |
| `internal/livegame/infiniteflight` | Hand-written IF Live API client (bearer auth) |
| `internal/livegame/jobs` | `register.go` job composition; `schedules/intervals.go` cron specs |
| `internal/livegame/scheduler` | robfig/cron wrapper with job metrics |
| `internal/community/*` | Users, VA, membership (+ roster, resolver), registration (+ route proof), user status |
| `internal/operations/{operator,reports}` | Platform operator workflows and report storage |
| `internal/cache` | `RedisStore`, `Store` interface, **all** keys in `keys.go` |
| `internal/metrics` | Single Prometheus `Registry` (HTTP, cache, jobs) |
| `internal/logging` | `log/slog` JSON handler (debug in local/development) |
| `migrations/` | Plain SQL, applied **manually** in filename order; no framework |

HTTP surfaces:

| Mount | Contract | Auth |
|---|---|---|
| `/health/status` | OpenAPI | public |
| `/metrics` | Prometheus | public on host (blocked by Caddy in prod) |
| `/api/v1/game/**` | OpenAPI | session cookie **or** `X-API-Key` |
| `/api/v1/signed-link` | OpenAPI | `X-API-Key` (+ `X-Discord-User-Id`) |
| `/api/v1/user/*`, `/memberships/join`, `/server/init`, `/reports/occupied-ifc`, `/admin/verify-god` | **hand-mounted, not in OpenAPI** | API key + `RequireDiscordBotContext` + `EnrichDiscordMembership` + registration rate limit |
| `/api/v1/operator/**` | **hand-mounted, not in OpenAPI** | session or API key + `RequirePlatformOperator` |
| `/auth/login`, `/auth/logout`, `/dashboard`, `/maps/**`, `/operator/**`, `/static/*` | SSR, never OpenAPI | `UISessionAuth` for dashboard group |

Roles (per VA membership): `proletariat` < `bourgeoisie` (staff) < `administrator`. Helpers in `internal/community/membership/roles.go` and `access/auth.Claims`.

Conventions that are enforced in code today (see `docs/politburo/conventions.md`):

- Cache-backed endpoints never call upstream/DB on miss — they return 503 and jobs own population.
- Handlers own cookies/sessions/claims; domain packages take primitive IDs only and must not import `internal/access/session`.
- Timestamps are `time.Time` until JSON, UTC, RFC 3339.
- Metrics are performance-only (HTTP, cache, jobs); no business gauges unless asked.

---

## Comrade Bot (`services/comrade-bot-discord`)

- Entry `src/index.ts`; client `src/bot/BotClient.ts`; routing `src/handlers/InteractionRouter.ts`.
- Commands in `src/commands/` (export `data` + `execute`), registered through `src/commands/registry.ts`; `src/configs/commandMap.ts` delegates to it.
- **All** Politburo HTTP in `src/services/apiService.ts`; headers from `src/helpers/utils.ts`; envelope parsing in `src/helpers/apiEnvelope.ts`.
- Config `src/configs/env.ts` (`API_URL` default `http://localhost:8080` — set `http://localhost:8082` locally).
- Observability: `src/infra/logger.ts` (JSON, redacts sensitive keys), `src/infra/metrics.ts` (`comrade_bot_*`), `src/infra/metricsServer.ts` (`/metrics`, `/healthz` on 9091).
- `npm run build` is the real typecheck; `npm test` runs `tests/*.test.ts` (logger/metrics only).

---

## Lookouts (known traps — check before relying on behavior)

**Contract / codegen**

1. Edit `openapi/politburo/**`, never `openapi/politburo.yaml`. Run `make generate` (or `make openapi-bundle`) and commit the bundle with the sources; CI runs `make openapi-bundle-check`.
2. New paths **and** new schemas must be listed in `openapi/politburo/openapi.yaml`; schema component names come from the file name.
3. Keep `generate-politburo` / `generate-infinite-flight` free of Node dependencies (Docker generate stage has none).
4. Politburo uses non-strict `chi-server`; new operations need a method on `apiHandler` in `transport/http/server.go` delegating to a handler — the compile error on `politburoapi.HandlerWithOptions` tells you when one is missing.
5. Identity/membership/operator JSON routes are hand-mounted and absent from OpenAPI. Adding them to the spec requires removing the manual mount and adding the adapter method — don't mount the same path twice.
6. Generated TS types (`src/generated/politburo-api.ts`) are **not used** by bot source yet; bot types in `src/types/Responses.ts` are hand-written.

**Bot ↔ Politburo drift**

7. `apiService.ts` still calls legacy endpoints the rewrite does not serve: `/api/v1/user/{ifcId}/flights`, `/api/v1/flights/va`, `/api/v1/pilot/stats`, `/api/v1/pireps/*`, `/api/v1/events/*`. Treat those commands as broken against the rewrite until ported.
8. Envelopes differ: Politburo returns `{data:...}` / `{error:{code,message}}`; legacy bot code expects `{status,result,message}`. `unwrapApiData` in `apiEnvelope.ts` accepts both, but error paths often still read `body.message`. Request bodies drift too (e.g. signed-link sends `redirect_to`/`ttl_minutes`; the spec field is `redirectTo`). Check the spec when touching a call.
9. Bot `API_URL` defaults to `:8080`; dev Politburo listens on `:8082`.

**Runtime / infra**

10. Port split: dev Politburo `8082` (config default, Dockerfile `EXPOSE`, dev Prometheus target `localhost:8082`); prod uses `PORT=8080` (compose `127.0.0.1:8080:8080`, prod Prometheus `politburo:8080`). Change both sides together.
11. Politburo health is `GET /health/status` (Docker/k8s probes and compose healthcheck use the same path at 15s).
12. `infra/prod/env/politburo.env.example` is legacy: lists `JWT_SECRET`, `GOD_MODE`, `USE_REDIS_CACHE`, `DEBUG` (unused) and omits `SIGNED_LINK_SECRET` (**required outside `APP_ENV=local`**), `UI_BASE_URL`, `PLATFORM_OPERATOR_DISCORD_IDS`, `JOBS_ENABLED`. It also defaults `PG_DB=infinite` while code defaults `politburo_next`.
13. Migrations are manual. Dev Compose creates DB `infinite`; `politburo_next` must be created and `000_core_schema.sql` applied by hand (see `infra/dev/README.md`). Legacy full schema is in `migrations/archive/`. Never apply `000` to a populated DB.
14. Promtail pipelines (dev+prod) parse Zap fields (`L`, `T`, `M`, `C`); Politburo now logs slog JSON (`time`, `level`, `msg`). `level` works; timestamp/message extraction does not.
15. Grafana dashboards in `infra/{dev,prod}/grafana/provisioning/dashboards/` are duplicated per env and mostly legacy: `politburo-background.json` queries metrics that no longer exist (queues, Watermill, webhooks, sync jobs, cache sizes), and HTTP panels use labels `endpoint`/`status_code` while the registry emits `route`/`status`. Current Politburo families: `politburo_http_*{method,route,status}`, `politburo_cache_*{operation,outcome}`, `politburo_jobs_*{job,outcome}`. Dev Loki streams use `service=`; prod Promtail tails `/var/log/containers/*.log` and labels `container_name=`.
16. `middleware/ratelimit.go` comments say "unwired" but it **is** mounted on the bot route group (registration limit, fails open).
17. `infra/prod/docker-compose.prod.yml` keeps project name `labour-bureau` on purpose (volume names). Don't rename.
18. Air must run with cwd `services/politburo`: `exec "$(cd ../../cicd/tools && go tool -n air)" -c .air.toml`. Running `go tool air` from `cicd/tools` builds the wrong tree.
19. In some agent sandboxes `$(MAKE)` resolves to a non-make binary; if `make` recursion aborts, run `/usr/bin/make MAKE=/usr/bin/make <target>`.
20. `internal/api/v1bot/` and `internal/db/migrations/` are empty leftovers — don't put code there.
