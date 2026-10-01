# Production operations

Canonical runbook: **[`infra/prod/README.md`](../../infra/prod/README.md)** (layout, env, systemd, compose commands).

Legacy Podman Compose remains under `infra/prod/` until k3s cutover ([`docs/politburo/future/production-kubernetes.md`](../politburo/future/production-kubernetes.md)).

Systemd units live in [`infra/prod/systemd/`](../../infra/prod/systemd/) (`compose-stack.service`, `caddy-rootful.service`, `podman-log-shipper.service`).
