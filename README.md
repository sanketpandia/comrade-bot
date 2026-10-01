# comrade-bot

Monorepo for the Infinite Experiment VA stack: Politburo (Go API + dashboard), the Discord bot, shared OpenAPI contracts, and infrastructure.

## Layout

| Path | Purpose |
|------|---------|
| `openapi/` | API contracts (Politburo + Infinite Flight client) |
| `cicd/` | Code generation Makefile and pinned Go tools |
| `services/politburo/` | Go backend |
| `services/comrade-bot-discord/` | Discord bot |
| `infra/dev/` | Local Docker Compose backing services |
| `infra/prod/` | Production Podman Compose and deploy scripts |
| `docs/` | Central documentation |

## Quick start

```sh
# From repo root
make generate

cd infra/dev
docker compose -f docker-compose.dev.yml up -d   # Postgres, Redis, observability, …

# Politburo (host, port 8082)
cd services/politburo && sh -c 'exec "$(cd ../../cicd/tools && go tool -n air)" -c .air.toml'

# Discord bot
cd services/comrade-bot-discord && API_URL=http://localhost:8082 npm run dev
```

Or use `infra/dev/start-dev.sh` for a tmux session with Compose + both services.

See [docs/README.md](docs/README.md) and [AGENTS.md](AGENTS.md) for detail.
