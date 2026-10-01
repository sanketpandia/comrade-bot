# Production operations

- **New server (Ubuntu 22.04):** [`ubuntu-22.04-production-bootstrap.md`](ubuntu-22.04-production-bootstrap.md)
- **Day-2 k3s:** [`infra/prod/k8s/README.md`](../../infra/prod/k8s/README.md)
- **Layout / legacy compose:** [`infra/prod/README.md`](../../infra/prod/README.md)

Legacy Podman Compose remains under `infra/prod/` for rollback ([`docs/politburo/future/production-kubernetes.md`](../politburo/future/production-kubernetes.md)).

Systemd units live in [`infra/prod/systemd/`](../../infra/prod/systemd/) (`compose-stack.service`, `caddy-rootful.service`, `podman-log-shipper.service`).
