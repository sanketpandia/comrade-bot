# AGENTS.md

## Workspace shape

Single monorepo (`comrade-bot` on GitHub). Run commands from the directory that owns the change, or use root `make` for codegen.

| Path | Role |
|------|------|
| `openapi/` | Politburo API sources (`politburo/`), bundled `politburo.yaml`, Infinite Flight upstream spec |
| `cicd/` | `make generate`; pinned Go tools (`oapi-codegen`, `air`) in `cicd/tools/`, Redocly in `cicd/package.json` |
| `services/politburo/` | Go API, jobs, embedded dashboard |
| `services/comrade-bot-discord/` | TypeScript Discord bot |
| `infra/dev/` | Local Compose backing services, observability config, `start-dev.sh` |
| `infra/prod/` | Production Podman Compose, Caddy, deploy scripts, env templates, Grafana |
| `_monorepo-backup/` | Bare backups of the old three repos — never edit |

## Dependencies

- Contract: `openapi/politburo/**` → (Redocly) `openapi/politburo.yaml` (committed) → oapi-codegen Go server + openapi-typescript bot types (both gitignored). Fresh checkouts need `make generate` before building.
- Politburo Docker builds run only the Go generators and have no Node, so they read the committed bundle.
- Runtime: bot → Politburo over HTTP (`X-API-Key`, `X-Discord-User-Id`, `X-Discord-Server-Id`); Politburo → Postgres, Redis, Infinite Flight API (jobs only). Prometheus scrapes Politburo and bot `/metrics`; Promtail → Loki → Grafana.
- Politburo wiring: `internal/app/app.go` (composition root), `internal/transport/http/server.go` (all routes), `internal/livegame/jobs/register.go` (all jobs).

Full map and known traps: `.claude/commands/architecture.md`. Agents in `.claude/agents/` (architect → swagger → developer → observability) read it first.

## Lookouts

- `docs/standards.md` and much of `docs/politburo/**`, `docs/infra/**` still describe the old multi-repo layout (`labour-bureau/`, `api/openapi/`, Vizburo, `internal/routes`). Verify against code.
- Identity/membership/operator `/api/v1` routes are hand-mounted in `server.go` and missing from OpenAPI; new JSON routes go through the spec.
- The bot still calls legacy endpoints the rewrite does not serve (`/healthCheck`, `/pireps/*`, `/events/*`, `/pilot/stats`, …), and doesn't use the generated TS types yet.
- Ports differ: dev Politburo `8082`, prod `8080`. Prod compose healthcheck still targets `/healthCheck`; prod `politburo.env.example` lacks `SIGNED_LINK_SECRET` (required outside local).
- Grafana dashboards and Promtail pipelines partly target legacy metrics/labels and Zap log keys; Politburo now logs `slog` JSON.
- Migrations are manual SQL; the `politburo_next` DB must be created by hand in dev.
- If `make` recursion aborts in an agent sandbox, run `/usr/bin/make MAKE=/usr/bin/make <target>`.

## Local stack

- Entry: `infra/dev/start-dev.sh` — Compose in tmux window 1; bot and Politburo on the host in windows 2–3.
- Backing services: `docker compose -f infra/dev/docker-compose.dev.yml up` from `infra/dev/`.
- Ports: Politburo `8082`, bot metrics `9091`, Postgres `5432`, Redis `6379`, Swagger UI `8081`, Prometheus `9090`, Grafana `3000`.

## Code generation

From repo root:

```sh
make generate              # bundle spec + all consumers
make openapi-bundle        # only rebuild openapi/politburo.yaml
make openapi-bundle-check  # fail if committed bundle is stale (CI)
make check                 # generate + Politburo tests
make openapi-view
```

Never hand-edit `services/politburo/internal/api/generated/**` or `services/comrade-bot-discord/src/generated/**`.

Politburo spec sources live in `openapi/politburo/` (one file per path and per schema). `openapi/politburo.yaml` is a committed Redocly bundle of them: edit the sources, run `make generate` (or `make openapi-bundle`), and commit both.

Docker builds use **repository root** as context:

```sh
docker build -f services/politburo/Dockerfile .
docker build -f services/comrade-bot-discord/Dockerfile .
```

## Politburo

- Work from `services/politburo/`.
- Hot reload: from `services/politburo`, `exec "$(cd ../../cicd/tools && go tool -n air)" -c .air.toml` (Air must run with cwd `services/politburo`; loads `.env` via `.air.toml`).
- Tests: `go test ./...` or `make test` in service dir.
- Migrations: `services/politburo/migrations/` (default DB `politburo_next`).

## Discord bot

- Work from `services/comrade-bot-discord/`; `npm run dev`, `npm run build`, `npm test`.
- `API_URL` defaults to `http://localhost:8080`; use `http://localhost:8082` for the rewrite.
- HTTP calls stay in `src/services/apiService.ts`.
- Slash command deploy: `npm run deploy:dev:local` with `GUILD_ID`.

## Production

- Deploy: `infra/prod/deploy-services.sh politburo|comrade-bot|all`
- Env templates: `infra/prod/env/*.env.example`

## Docs

Dev notes: `docs/` (verify older pages against code). Politburo product docs under `docs/politburo/`. OpenAPI flow: `docs/development/openapi.md`.
