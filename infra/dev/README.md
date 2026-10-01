# Local development infrastructure

Docker Compose backing services for the comrade-bot monorepo. Politburo and the Discord bot run on the host for live reload.

## Prerequisites

- Docker (or Podman with `CONTAINER_CLI=podman`)
- Go 1.25+ (`services/politburo`)
- Node 20+ (`services/comrade-bot-discord`)
- tmux (for `start-dev.sh`)

## First-time setup

**1. Code generation (repo root)**

```sh
make generate
```

**2. Politburo**

```sh
cd services/politburo
cp .env.example .env
```

Use `PORT=8082`, `PG_HOST=localhost`, `PG_DB=politburo_next`, and `IF_API_KEY` when jobs are enabled.

**3. Database**

From `infra/dev/`:

```sh
docker compose -f docker-compose.dev.yml up -d db
docker compose -f docker-compose.dev.yml exec -T db \
  psql -U ieuser -d postgres -c "CREATE DATABASE politburo_next;" 2>/dev/null || true
docker compose -f docker-compose.dev.yml exec -T db \
  psql -v ON_ERROR_STOP=1 -1 -U ieuser -d politburo_next \
  < ../../services/politburo/migrations/000_core_schema.sql
```

**4. Discord bot**

```sh
cd services/comrade-bot-discord
npm ci
```

Set `DISCORD_BOT_TOKEN`, `DISCORD_BOT_CLIENT_ID`, and `API_URL=http://localhost:8082`.

## Daily flow

```sh
./start-dev.sh
```

tmux session `infinite-stage`: Compose (window 1), `npm run dev` (window 2), Air (window 3).

Politburo logs: `/tmp/politburo.log` (Promtail → Loki).

## Compose services

| Service | Host port |
|---------|-----------|
| Postgres | 5432 |
| Redis | 6379 |
| pgAdmin | 5050 |
| Swagger UI | 8081 |
| Prometheus | 9090 |
| Grafana | 3000 |
| Loki | 3100 |

Further detail: [docs/README.md](../../docs/README.md), [docs/development/openapi.md](../../docs/development/openapi.md).
