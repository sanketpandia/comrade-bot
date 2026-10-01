# Systemd (production VPS)

Use the units in [`infra/prod/systemd/`](../../infra/prod/systemd/). Edit `WorkingDirectory`, volume paths, `User`, `Group`, and `XDG_RUNTIME_DIR` before installing.

| Unit | Purpose |
|------|---------|
| `compose-stack.service` | `podman compose up` for the full stack (foreground; `Restart=always`) |
| `caddy-rootful.service` | Caddy on host network with [`edge/Caddyfile`](../../infra/prod/edge/Caddyfile) |
| `podman-log-shipper.service` | Tails container logs into `/var/log/containers` for Promtail |

Install:

```bash
cd infra/prod
sudo cp systemd/compose-stack.service systemd/caddy-rootful.service systemd/podman-log-shipper.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now compose-stack.service caddy-rootful.service podman-log-shipper.service
```

Logs: `sudo journalctl -u compose-stack.service -f`

Full setup: [`infra/prod/README.md`](../../infra/prod/README.md).
