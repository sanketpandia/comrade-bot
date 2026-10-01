# Production (`infra/prod`)

Configuration for a **single VPS**: Podman Compose stack (interim) or **k3s** (planned). Application images come from **GHCR** on `main`; avoid on-host `docker build` for politburo and the Discord bot on new installs.

## Layout

```
infra/prod/
├── docker-compose.prod.yml   # Full stack: apps, Postgres, Redis, observability
├── env/                      # Secret templates → copy to *.env on the server (gitignored)
├── scripts/                  # Compose helpers (deploy, log shipper, cleanup)
├── observability/            # Prometheus, Loki, Promtail, Grafana provisioning
├── edge/                     # Caddyfile + deploy script; ACME dirs gitignored
├── systemd/                  # compose-stack, caddy-rootful, podman-log-shipper units
├── start-services.sh         # Ad-hoc compose up
├── stop-services.sh
└── deploy-services.sh        # Rebuild politburo and/or comrade-bot via compose
```

| Path | Role |
|------|------|
| [`env/`](env/) | Copy `*.env.example` → `*.env`; `chmod 600` |
| [`observability/`](observability/) | Metrics/logs dashboards; mounted by compose |
| [`edge/Caddyfile`](edge/Caddyfile) | Public TLS for API + Grafana |
| [`systemd/compose-stack.service`](systemd/compose-stack.service) | Keeps `podman compose up` running under systemd |
| [`systemd/caddy-rootful.service`](systemd/caddy-rootful.service) | Caddy on host network → `127.0.0.1:8080` / `:3000` |
| [`systemd/podman-log-shipper.service`](systemd/podman-log-shipper.service) | Writes container logs for Promtail |

Future k8s manifests will live under `infra/prod/k8s/` (see [`docs/politburo/future/production-kubernetes.md`](../../docs/politburo/future/production-kubernetes.md)).

## First-time setup

1. **Env files** — from `infra/prod/env/`:

   ```bash
   cp politburo.env.example politburo.env
   cp comrade-bot.env.example comrade-bot.env
   cp database.env.example database.env
   cp cache.env.example cache.env
   cp monitoring.env.example monitoring.env
   ```

   Replace every `CHANGE_ME` with strong secrets (`openssl rand -base64 32`).

2. **Persistent data** — point Podman volume storage at your mounted disk if needed ([Podman storage](https://docs.podman.io/en/latest/markdown/podman-system.1.html)).

3. **Systemd** — edit paths in [`systemd/*.service`](systemd/) (`WorkingDirectory`, `User`, `XDG_RUNTIME_DIR`), then:

   ```bash
   sudo cp systemd/compose-stack.service systemd/caddy-rootful.service systemd/podman-log-shipper.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable --now compose-stack.service caddy-rootful.service podman-log-shipper.service
   ```

## Day-to-day

| Task | Command |
|------|---------|
| Start stack (manual) | `./start-services.sh` |
| Stop stack | `./stop-services.sh` |
| Deploy app rebuild | `./deploy-services.sh politburo` · `comrade-bot` · `all` |
| Reload Caddy | `./edge/deploy-caddy.sh` |
| Compose status | `podman compose -f docker-compose.prod.yml ps` |
| Stuck “name in use” | `./scripts/clean-exited-for-compose.sh` then start again |

**CI images:** `ghcr.io/<github-owner>/politburo:<sha>` and `comrade-bot:<sha>` (see [`docs/politburo/development/containers.md`](../../docs/politburo/development/containers.md)).

## Public endpoints

- `https://comradebot.cc` → politburo (`127.0.0.1:8080`)
- `https://monitor.comradebot.cc` → Grafana (`127.0.0.1:3000`)

Prometheus, Loki, Redis, and the Discord bot are not published to the internet.

## Backups

```bash
podman compose -f docker-compose.prod.yml exec db pg_dump -U ieuser infinite > backup.sql
```

Redis persists in the `redis-prod` volume (AOF).
