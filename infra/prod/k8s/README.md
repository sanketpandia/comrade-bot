# k3s production (lalquila)

Single-node **k3s** stack for politburo, comrade-bot, Postgres, Redis, and observability. Images come from **GHCR**; deploys run on the **self-hosted** GitHub Actions runner on the VPS.

**Day-2 ops (secrets, migrations, restarts, `git pull`):** [`docs/infra/prod-ops.md`](../../../docs/infra/prod-ops.md).

**New server from scratch?** Follow [`docs/infra/ubuntu-22.04-production-bootstrap.md`](../../../docs/infra/ubuntu-22.04-production-bootstrap.md) (Ubuntu 22.04, DNS, k3s, Caddy, runner, secrets, smoke tests).

## Layout

```
infra/prod/k8s/
├── base/                 # Namespaces, workloads, Services
│   ├── data/             # Postgres + Redis (ie-data)
│   ├── observability/    # Prometheus, Loki, Grafana, Promtail (ie-observability)
│   └── apps/             # politburo + comrade-bot (ie-apps)
└── overlays/prod/        # ConfigMaps from infra/prod/observability + apply entrypoint
```

| Namespace | Services |
|-----------|----------|
| `ie-data` | `postgres`, `redis` |
| `ie-apps` | `politburo` (:8080 on `127.0.0.1` via hostPort), `comrade-bot` |
| `ie-observability` | `prometheus`, `loki`, `grafana` (:3000 on `127.0.0.1`), `promtail`, `cadvisor`, `node-exporter` DaemonSets |

Host **Caddy** (native `caddy.service`) proxies public HTTPS to `127.0.0.1:8080` and `127.0.0.1:3000` — sync [`../edge/Caddyfile`](../edge/Caddyfile) to `/etc/caddy/Caddyfile`.

## Prerequisites

- Ubuntu 22.04+ VPS (lalquila), ≥ 4 GB RAM (8 GB recommended)
- Mounted data disk (e.g. Hetzner `/mnt/HC_Volume_*`)
- Ports **80/443** for Caddy; **do not** expose k3s API (6443) publicly
- GitHub repo secrets (see [CI/CD secrets](#cicd-secrets))

## 1. Install k3s

```bash
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server \
  --disable traefik \
  --write-kubeconfig-mode 644" sh -

sudo kubectl get nodes
```

### PVC data on mounted disk

Default **local-path** stores under `/var/lib/rancher/k3s/storage`. To use a mounted volume:

```bash
# Example: point local-path at Hetzner volume (adjust path)
sudo mkdir -p /mnt/HC_Volume_12345/k3s-storage
sudo mkdir -p /etc/rancher/k3s
sudo tee /etc/rancher/k3s/config.yaml <<'EOF'
kubelet-arg:
  - "root-dir=/var/lib/kubelet"
EOF
# Follow k3s docs to patch local-path-provisioner helper pod path to /mnt/HC_Volume_12345/k3s-storage
```

Verify PVCs bind before relying on production data.

## 2. Self-hosted runner (`gitrunner` user)

1. GitHub → **Settings → Actions → Runners → New self-hosted runner** (Linux x64).
2. Install under `/home/gitrunner/actions-runner` as user **`gitrunner`** (dedicated service account; not your normal SSH login if that is also named `runner`).
3. Labels: add **`prod`** and **`linux`** at registration (`--labels prod,linux`); GitHub also adds `self-hosted`, `Linux`, `X64`.
4. **Kubeconfig** for the runner:

```bash
sudo mkdir -p /home/gitrunner/.kube
sudo cp /etc/rancher/k3s/k3s.yaml /home/gitrunner/.kube/config
sudo chown -R gitrunner:gitrunner /home/gitrunner/.kube
sudo chmod 600 /home/gitrunner/.kube/config
```

Optional: use GitHub secret `KUBECONFIG_B64` instead (deploy workflow writes `~/.kube/config` per job).

### Runner hooks (optional)

```bash
# /etc/systemd/system/actions.runner.*.service.d/override.conf
[Service]
Environment=ACTIONS_RUNNER_HOOK_JOB_STARTED=/home/gitrunner/hooks/job-started.sh
```

`job-started.sh`: `umask 077`, verify `kubectl cluster-info` — **no secrets in hook scripts**.

## 3. Kubernetes secrets (bootstrap once)

Create secrets **before** app pods start. Copy [`../env/*.env.example`](../env/) → `../env/*.env` on the server (`chmod 600`; `*.env` is gitignored). For k3s, put cluster DNS in app env files (see [prod-ops § Secrets](../../../docs/infra/prod-ops.md)) instead of relying on the patches below long term.

```bash
cd /path/to/comrade-bot

# Postgres (keys must match StatefulSet)
kubectl create namespace ie-data --dry-run=client -o yaml | kubectl apply -f -
kubectl -n ie-data create secret generic postgres-credentials \
  --from-literal=POSTGRES_DB=infinite \
  --from-literal=POSTGRES_USER=ieuser \
  --from-literal=POSTGRES_PASSWORD='…'

kubectl -n ie-data create secret generic redis-credentials \
  --from-literal=REDIS_PASSWORD='…'

# Politburo — use cluster DNS for data tier
kubectl create namespace ie-apps --dry-run=client -o yaml | kubectl apply -f -
kubectl -n ie-apps create secret generic politburo-env \
  --from-env-file=../env/politburo.env
kubectl -n ie-apps patch secret politburo-env --type merge -p \
  '{"stringData":{"PG_HOST":"postgres.ie-data.svc.cluster.local","REDIS_HOST":"redis.ie-data.svc.cluster.local"}}'

# Comrade-bot — in-cluster API
kubectl -n ie-apps create secret generic comrade-bot-env \
  --from-env-file=../env/comrade-bot.env
kubectl -n ie-apps patch secret comrade-bot-env --type merge -p \
  '{"stringData":{"API_URL":"http://politburo.ie-apps.svc.cluster.local:8080"}}'

# Grafana
kubectl create namespace ie-observability --dry-run=client -o yaml | kubectl apply -f -
kubectl -n ie-observability create secret generic grafana-credentials \
  --from-literal=GRAFANA_ADMIN_PASSWORD='…'

# GHCR pull (private packages)
kubectl -n ie-apps create secret docker-registry ghcr-cred \
  --docker-server=ghcr.io \
  --docker-username=GITHUB_USER \
  --docker-password=GITHUB_PAT_WITH_read_packages
```

Edit [`base/kustomization.yaml`](base/kustomization.yaml) `images.newName` if your GHCR owner is not `infinite-experiment`.

## 4. Apply manifests

From repo root (or clone on the server):

```bash
bash infra/prod/k8s/apply.sh
```

(`apply.sh` uses `kustomize build --load-restrictor LoadRestrictionsNone` so ConfigMaps can source files under `infra/prod/observability/`.)

## 5. Migrate Postgres from Podman

While Podman `db` still runs on `127.0.0.1:5432`:

```bash
podman exec db pg_dump -U ieuser -Fc infinite > /tmp/infinite.dump

# After cluster Postgres is Ready:
kubectl -n ie-data exec -it postgres-0 -- pg_restore -U ieuser -d infinite --clean --if-exists < /tmp/infinite.dump
# Or copy dump into pod and pg_restore locally
```

Apply SQL migrations if needed (see [`docs/infra/prod-ops.md`](../../../docs/infra/prod-ops.md) — run **individual** files, not blind loops on existing DBs):

```bash
for f in services/politburo/migrations/*.sql; do
  kubectl -n ie-data exec -i postgres-0 -- psql -U ieuser -d infinite -f - < "$f"
done
```

When verified:

```bash
podman stop db redis
podman rm db redis
```

## 6. Caddy (native systemd)

```bash
sudo cp infra/prod/edge/Caddyfile /etc/caddy/Caddyfile
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

Preserve existing ACME data under `/var/lib/caddy` (or your current path).

## 7. Smoke tests

- `curl -sS http://127.0.0.1:8080/health/status`
- `https://comradebot.cc/public/…` via Caddy
- Discord bot responds
- `https://monitor.comradebot.cc` → Grafana
- Prometheus targets: politburo + comrade-bot UP (in-cluster)
- Grafana → Explore → Loki: `{namespace="ie-apps"}` or `{container_name="politburo"}` returns lines

### Loki has no pod logs

Promtail must discover pods **on this node** and read `/var/log/pods` on the host. On a fresh cluster the usual causes are:

1. **Missing `HOSTNAME` env** — Promtail 3.x uses `$HOSTNAME` when filtering Kubernetes SD targets. Without `spec.nodeName` downward API, it defaults to the Promtail **pod** name, matches no node, and ships nothing (DaemonSet still looks “healthy”).
2. **Missing `__host__` relabel** — Each target needs `__host__` from `__meta_kubernetes_pod_node_name` (see [Promtail scraping docs](https://grafana.com/docs/loki/latest/send-data/promtail/scraping/#kubernetes-discovery)).
3. **Host log permissions** — Promtail runs as root in our manifests so it can read kubelet log files under `/var/log/pods`.
4. **Grafana query labels** — Provisioned dashboards filter `{container_name="politburo"}`. k8s Promtail also sets `namespace`, `pod`, `container`, and `service`.

Quick checks on the VPS:

```bash
kubectl -n ie-observability logs daemonset/promtail --tail=80
kubectl -n ie-observability exec daemonset/promtail -- sh -c \
  'echo HOSTNAME=$HOSTNAME; ls /var/log/pods | head -3'
# Loki image has no curl/wget; query from the host via port-forward:
kubectl -n ie-observability port-forward svc/loki 3100:3100 >/tmp/loki-pf.log 2>&1 &
PF_PID=$!
sleep 2
curl -sS http://127.0.0.1:3100/loki/api/v1/labels
kill "$PF_PID" 2>/dev/null || true
```

After fixing manifests, re-apply and restart Promtail:

```bash
cd /opt/comrade-bot && bash infra/prod/k8s/apply.sh
kubectl -n ie-observability rollout restart daemonset/promtail
```

## CD / CI secrets

| Secret | Purpose |
|--------|---------|
| `DISCORD_WEBHOOK_URL` | CI/CD Discord notifications |
| `GHCR_PULL_TOKEN` | Optional PAT to refresh `ghcr-cred` in deploy jobs |
| `KUBECONFIG_B64` | Optional; deploy job writes kubeconfig instead of on-disk file |

Deploy jobs use GitHub **Environment** `production` and `runs-on: [self-hosted, linux, prod]`.

**Discord slash commands** are not updated on every deploy. Opt in via merge commit `[sync-cmds]` or **Actions → CI/CD → Run workflow** (see [`docs/discord-bot/DEPLOYMENT.md`](../../../docs/discord-bot/DEPLOYMENT.md)). The `sync_discord_commands` job runs [`../scripts/k8s-sync-discord-commands.sh`](../scripts/k8s-sync-discord-commands.sh) on the prod runner.

## Operations

```bash
kubectl -n ie-apps get deploy,pods,svc
kubectl -n ie-apps rollout status deployment/politburo
kubectl -n ie-data logs postgres-0
kubectl -n ie-observability port-forward svc/prometheus 9090:9090
```

Rollback image:

```bash
kubectl -n ie-apps set image deployment/politburo politburo=ghcr.io/OWNER/politburo:PREVIOUS_SHA
kubectl -n ie-apps rollout status deployment/politburo
```

## Notifications

See [`../../../docs/infra/ci-notifications.md`](../../../docs/infra/ci-notifications.md).
