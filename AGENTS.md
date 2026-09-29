# AGENTS.md

## Workspace shape

Single monorepo (`comrade-bot` on GitHub). Run commands from the directory that owns the change, or use root `make` for codegen.

| Path | Role |
|------|------|
| `openapi/` | Politburo public API + Infinite Flight upstream spec |
| `cicd/` | `make generate`, pinned `oapi-codegen` in `cicd/tools/` |
| `services/politburo/` | Go API, jobs, embedded dashboard |
| `services/comrade-bot-discord/` | TypeScript Discord bot |
| `infra/dev/` | Local Compose backing services only |
| `infra/prod/` | Production Podman Compose and deploy scripts |

## Local stack

- Entry: `infra/dev/start-dev.sh` — Compose in tmux window 1; bot and Politburo on the host in windows 2–3.
- Backing services: `docker compose -f infra/dev/docker-compose.dev.yml up` from `infra/dev/`.
- Ports: Politburo `8082`, bot metrics `9091`, Postgres `5432`, Redis `6379`, Swagger UI `8081`, Prometheus `9090`, Grafana `3000`.

## Code generation

From repo root:

```sh
make generate    # all consumers
make check       # generate + Politburo tests
make openapi-view
```

Never hand-edit `services/politburo/internal/api/generated/**` or `services/comrade-bot-discord/src/generated/**`.

Docker builds use **repository root** as context:

```sh
docker build -f services/politburo/Dockerfile .
docker build -f services/comrade-bot-discord/Dockerfile .
```

## Politburo

- Work from `services/politburo/`.
- Hot reload: from `services/politburo`, `cd ../../cicd/tools && go tool air -c ../../services/politburo/.air.toml` (Air loads `services/politburo/.env` via `.air.toml`).
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

Authoritative dev notes: `docs/`. Politburo product docs under `docs/politburo/`. OpenAPI flow: `docs/development/openapi.md`.
