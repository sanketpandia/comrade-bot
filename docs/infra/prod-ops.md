# Production operations (lalquila / k3s)

Day-2 runbook for the primary stack: **k3s**, **GHCR images**, **GitHub Actions** on the self-hosted runner. Legacy Podman Compose lives under [`infra/prod/`](../../infra/prod/README.md) for rollback only.

**First-time server:** [`ubuntu-22.04-production-bootstrap.md`](ubuntu-22.04-production-bootstrap.md) → then [`infra/prod/k8s/README.md`](../../infra/prod/k8s/README.md) for bootstrap secrets and apply.

| Namespace | Workloads |
|-----------|-----------|
| `ie-data` | Postgres (`postgres-0`), Redis |
| `ie-apps` | Politburo (hostPort `127.0.0.1:8080`), comrade-bot |
| `ie-observability` | Prometheus, Loki, Grafana (hostPort `127.0.0.1:3000`), Promtail, … |

**Edge:** native host `caddy.service` → `127.0.0.1:8080` (Politburo), `127.0.0.1:3000` (Grafana). Public: `https://comradebot.cc`, `https://monitor.comradebot.cc`.

---

## When to `git pull` on the server

Routine **application** deploys do **not** require a pull in `/opt/comrade-bot`. On push to `main`, CI builds images, pushes to GHCR, and runs `kubectl set image` on the prod runner (fresh checkout each job).

| You need… | Pull `/opt/comrade-bot`? | Who applies it |
|-----------|-------------------------|----------------|
| New Politburo / bot **code** (normal merge to `main`) | **No** | CI: image build + rollout |
| Changes under `infra/prod/k8s/**` | **No** (usually) | CI job `apply_infra` runs `infra/prod/k8s/apply.sh` |
| **SQL migration** from repo | **Yes** — before `psql … < services/politburo/migrations/…` | You, on the server |
| Manual `bash infra/prod/k8s/apply.sh` | **Yes** — if CI did not run or you are fixing drift | You |
| Updated [`infra/prod/edge/Caddyfile`](../../infra/prod/edge/Caddyfile) | **Yes** — then copy to `/etc/caddy/` and reload | You |
| **Secrets** in `infra/prod/env/*.env` | **N/A** — those files are **not in git** (see below) | You → `kubectl` secret sync |

Keep a clone at e.g. `/opt/comrade-bot` for migrations, manual apply, and Caddy sync. It does not need to track `main` for every app release.

---

## What CI does *not* do

- Does **not** read or update `politburo-env` / `comrade-bot-env` from the repo.
- Does **not** run Postgres migrations.
- Does **not** reload Caddy when only app images change.

GitHub **environment** `production` secrets (`KUBECONFIG_B64`, `GHCR_PULL_TOKEN`, Discord webhooks) are for **CI only**, not injected into app pods.

---

## Secrets and env files

Templates (committed): `infra/prod/env/*.env.example`.  
Live values (server only, gitignored): `infra/prod/env/*.env` — `chmod 600`.

Kubernetes reads **secrets**, not the files directly:

| Secret | Namespace | Source file (typical) |
|--------|-----------|------------------------|
| `postgres-credentials`, `redis-credentials` | `ie-data` | `database.env`, `cache.env` (bootstrap) |
| `politburo-env` | `ie-apps` | `politburo.env` |
| `comrade-bot-env` | `ie-apps` | `comrade-bot.env` |
| `grafana-credentials` | `ie-observability` | `monitoring.env` (bootstrap) |

### k3s hostnames in app env files

On k3s, set these in **`politburo.env`** / **`comrade-bot.env`** so you do not need post-create `kubectl patch`:

```env
# politburo.env
PG_HOST=postgres.ie-data.svc.cluster.local
REDIS_HOST=redis.ie-data.svc.cluster.local

# comrade-bot.env (exactly one API_URL line — duplicates break kubectl)
API_URL=http://politburo.ie-apps.svc.cluster.local:8080
```

Politburo also needs production config the templates may omit or under-document; verify against [`services/politburo/.env.example`](../../services/politburo/.env.example), especially **`SIGNED_LINK_SECRET`** (required when `APP_ENV=production`), **`UI_BASE_URL`** (e.g. `https://comradebot.cc`), and **`PG_DB=infinite`** to match cluster Postgres.

### Sync file → secret → pods

From repo root on the server:

```bash
cd /opt/comrade-bot/infra/prod/k8s

kubectl -n ie-apps create secret generic politburo-env \
  --from-env-file=../env/politburo.env \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n ie-apps create secret generic comrade-bot-env \
  --from-env-file=../env/comrade-bot.env \
  --dry-run=client -o yaml | kubectl apply -f -
```

Running pods do not reload secrets. Restart after any secret change:

```bash
kubectl -n ie-apps rollout restart deployment/politburo
kubectl -n ie-apps rollout status deployment/politburo --timeout=5m

kubectl -n ie-apps rollout restart deployment/comrade-bot
kubectl -n ie-apps rollout status deployment/comrade-bot --timeout=5m
```

If `kubectl create secret … --from-env-file` errors with **duplicate key** (e.g. two `API_URL=` lines), fix the file and re-run apply.

Initial bootstrap (first install): [`infra/prod/k8s/README.md` § Kubernetes secrets](../../infra/prod/k8s/README.md).

---

## Restarts and rollouts

| Action | Command |
|--------|---------|
| Status | `kubectl -n ie-apps get deploy,pods` |
| Wait for Politburo | `kubectl -n ie-apps rollout status deployment/politburo --timeout=5m` |
| Manual image (emergency) | `kubectl -n ie-apps set image deployment/politburo politburo=ghcr.io/OWNER/politburo:SHA` |
| Logs | `kubectl -n ie-apps logs -l app=politburo --tail=100` |
| Previous crash | `kubectl -n ie-apps logs -l app=politburo --previous --tail=100` |
| Apply k8s manifests | `bash infra/prod/k8s/apply.sh` (or wait for CI on `infra/prod/k8s/**` push) |
| Reload Caddy | `sudo caddy validate --config /etc/caddy/Caddyfile && sudo systemctl reload caddy` |

Politburo readiness: `GET /health/status` on port 8080. Stuck rollout → check pod events and logs first (`describe pod`, `logs`).

**Discord slash commands** are not updated every deploy. Opt in: merge commit message `[sync-cmds]` or **Actions → CI/CD → Run workflow**. See [`docs/discord-bot/DEPLOYMENT.md`](../discord-bot/DEPLOYMENT.md).

---

## Database operations

- **Database name:** `infinite` (must match `PG_DB` in `politburo-env`).
- **Migrations:** manual SQL; Politburo does **not** migrate on startup. See [`services/politburo/migrations/README.md`](../../services/politburo/migrations/README.md).

### Backup

```bash
kubectl -n ie-data exec postgres-0 -- \
  pg_dump -U ieuser -Fc infinite > "/tmp/infinite-$(date +%Y%m%d-%H%M).dump"
```

### Apply a new migration

After `git pull` on the server (to get the new `.sql` file):

```bash
cd /opt/comrade-bot

kubectl -n ie-data exec -i postgres-0 -- \
  psql -v ON_ERROR_STOP=1 -U ieuser -d infinite -f - \
  < services/politburo/migrations/001_game_aircraft_liveries.sql
```

Run **only** migration files that have not been applied yet. Do **not** re-run `000_core_schema.sql` on an existing production database.

### Restore / sanity

```bash
kubectl -n ie-data get pods
kubectl -n ie-data exec postgres-0 -- pg_isready -U ieuser -d infinite
```

---

## Observability

- Grafana: `https://monitor.comradebot.cc` (via Caddy → `127.0.0.1:3000`).
- Port-forward Prometheus: `kubectl -n ie-observability port-forward svc/prometheus 9090:9090`.
- CI notifications: [`ci-notifications.md`](ci-notifications.md).

ConfigMap-only manifest changes: `apply.sh` restarts Prometheus and Grafana in `ie-observability`; Promtail may need a manual rollout restart if log paths change (see k8s README).

---

## Related docs

| Topic | Doc |
|-------|-----|
| k8s bootstrap, PVC, CI secrets | [`infra/prod/k8s/README.md`](../../infra/prod/k8s/README.md) |
| Layout, legacy compose | [`infra/prod/README.md`](../../infra/prod/README.md) |
| Containers / GHCR tags | [`docs/politburo/development/containers.md`](../politburo/development/containers.md) |
