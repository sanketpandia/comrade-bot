# Production (`infra/prod`)

Configuration for a **single VPS** (lalquila). **Primary:** k3s + GHCR images + self-hosted runner CD. **Legacy:** Podman Compose for ad-hoc or rollback.

Application images: `ghcr.io/<github-owner>/politburo:<sha>` and `comrade-bot:<sha>` on `main` (see [`docs/politburo/development/containers.md`](../../docs/politburo/development/containers.md)).

**Fresh Ubuntu 22.04 VPS:** [`docs/infra/ubuntu-22.04-production-bootstrap.md`](../../docs/infra/ubuntu-22.04-production-bootstrap.md).

## k3s (primary)

Manifests and runbook: **[`k8s/README.md`](k8s/README.md)**.

| Item | Detail |
|------|--------|
| Apply | `bash k8s/apply.sh` |
| Namespaces | `ie-data`, `ie-apps`, `ie-observability` |
| Edge | **Native** `caddy.service` on the host → `127.0.0.1:8080` / `:3000` (sync [`edge/Caddyfile`](edge/Caddyfile) to `/etc/caddy/Caddyfile`) |
| CD | GitHub Actions `deploy` jobs on runner labels `self-hosted`, `linux`, `prod`; environment **`production`** |
| Notifications | [`docs/infra/ci-notifications.md`](../../docs/infra/ci-notifications.md) (`DISCORD_WEBHOOK_URL`) |

On **lalquila**, Caddy is already installed as rootful **systemd** (`/usr/bin/caddy`). Do not use [`systemd/caddy-rootful.service`](systemd/caddy-rootful.service) (Podman Caddy) unless you intentionally switch edge models.

## Layout

```
infra/prod/
├── k8s/                      # Kustomize: k3s workloads (primary)
├── docker-compose.prod.yml   # Legacy full stack
├── env/                      # Secret templates → *.env on server / kubectl secrets
├── scripts/                  # Compose deploy helpers
├── observability/            # Prometheus, Loki, Promtail, Grafana (compose + k8s ConfigMaps)
├── edge/                     # Caddyfile + deploy-caddy.sh
├── systemd/                  # compose-stack, caddy-rootful (Podman), podman-log-shipper
├── start-services.sh
├── stop-services.sh
└── deploy-services.sh        # On-host rebuild via compose (avoid for prod apps; use GHCR)
```

## Legacy Compose (interim)

| Path | Role |
|------|------|
| [`env/`](env/) | Copy `*.env.example` → `*.env`; `chmod 600` |
| [`systemd/compose-stack.service`](systemd/compose-stack.service) | `podman compose up` under systemd |
| [`systemd/podman-log-shipper.service`](systemd/podman-log-shipper.service) | Podman logs for Promtail (not used on k3s) |

```bash
cp env/*.env.example → env/*.env   # fill secrets
./start-services.sh               # or enable compose-stack.service
```

## Day-to-day (k3s)

| Task | Command |
|------|---------|
| Rollout (automatic) | Push to `main` (path-filtered workflows) |
| Manual image rollout | `kubectl -n ie-apps set image deployment/politburo politburo=ghcr.io/OWNER/politburo:SHA` |
| Manifest apply | `bash k8s/apply.sh` or workflow **Deploy k8s manifests** |
| Reload Caddy | `sudo caddy validate --config /etc/caddy/Caddyfile && sudo systemctl reload caddy` |
| Cluster status | `kubectl -n ie-apps get deploy,pods` |

## Public endpoints

- `https://comradebot.cc` → politburo (`127.0.0.1:8080`)
- `https://monitor.comradebot.cc` → Grafana (`127.0.0.1:3000`)

## Backups

**k3s Postgres:**

```bash
kubectl -n ie-data exec postgres-0 -- pg_dump -U ieuser infinite > backup.sql
```

**Legacy compose:** `podman compose -f docker-compose.prod.yml exec db pg_dump …`
