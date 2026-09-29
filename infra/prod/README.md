# Production Deployment Guide

This directory contains all production deployment configuration files for the Infinite Experiment stack.

## Directory Structure

```
prod/
├── docker-compose.prod.yml          # Main production compose file
├── labour-bureau-compose.service    # Systemd unit (recommended: keeps stack up reliably)
├── labour-bureau-services.service   # DEPRECATED: use labour-bureau-compose.service
├── start-services.sh                # Manual start via podman compose
├── stop-services.sh                 # Manual stop via podman compose
├── deploy-services.sh              # Deploy politburo and/or comrade-bot
├── Caddyfile                        # Caddy reverse proxy config
├── prometheus.prod.yml              # Prometheus config
├── loki.prod.yml                    # Loki config
├── promtail-config.yml              # Promtail config
├── grafana/                         # Grafana provisioning
├── env/                             # Environment variable templates
│   ├── politburo.env.example
│   ├── comrade-bot.env.example
│   ├── database.env.example
│   ├── cache.env.example
│   └── monitoring.env.example
└── README.md                        # This file
```

## Initial Setup

### 1. Create Environment Files

Copy each `.example` file to create the actual environment files (without `.example`):

```bash
cd prod/env
cp politburo.env.example politburo.env
cp comrade-bot.env.example comrade-bot.env
cp database.env.example database.env
cp cache.env.example cache.env
cp monitoring.env.example monitoring.env
```

### 2. Generate Secure Secrets

**Generate strong passwords for each service:**

```bash
# PostgreSQL password
openssl rand -base64 32

# Redis password
openssl rand -base64 32

# JWT secret
openssl rand -base64 32

# Grafana admin password
openssl rand -base64 32
```

### 3. Configure Environment Files

Edit each `.env` file and replace all `CHANGE_ME` values with your generated secrets:

- **`database.env`**: Set `POSTGRES_PASSWORD`
- **`cache.env`**: Set `REDIS_PASSWORD`
- **`politburo.env`**: Set `PG_PASSWORD`, `REDIS_PASSWORD`, `IF_API_KEY`, `GOD_MODE`, `JWT_SECRET`
- **`comrade-bot.env`**: Set `BOT_TOKEN` (Discord bot token)
- **`monitoring.env`**: Set `GRAFANA_ADMIN_PASSWORD`

### 4. Configure Podman to Use Mounted Disk for Volumes

To store all Podman volumes (including PostgreSQL data) on your mounted disk (`/mnt/HC_Volume_104770220`), configure Podman's data-root (e.g. in `/etc/containers/storage.conf` or via `--root`). See [Podman storage](https://docs.podman.io/en/latest/markdown/podman-system.1.html).

**Note:** After changing the data-root, existing volumes will need to be migrated or recreated. If you have existing volumes, you may need to:
1. Stop all containers: `podman compose -f docker-compose.prod.yml down`
2. Backup existing volumes (if any)
3. Restart Podman / adjust storage config
4. Recreate volumes: `podman compose -f docker-compose.prod.yml up -d`

### 5. Security Notes

⚠️ **IMPORTANT**: 
- Never commit actual `.env` files to git (they should be in `.gitignore`)
- Keep `.env.example` files as templates only
- Use strong, unique passwords for each service
- Restrict file permissions: `chmod 600 prod/env/*.env`

## Deployment

### Reliable startup with systemd (Podman, recommended)

For production on Podman (e.g. Ubuntu), use the **labour-bureau-compose.service** systemd unit so the stack stays up across reboots and systemd accurately reflects container state:

1. Edit `labour-bureau-compose.service` and set `WorkingDirectory`, `User`, `Group`, `EnvironmentFile`, and `XDG_RUNTIME_DIR` to match your server (paths and uid).
2. Install and enable:
   ```bash
   sudo cp prod/labour-bureau-compose.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable --now labour-bureau-compose.service
   ```
3. The unit runs `podman compose up` in the foreground; if it exits, systemd restarts it. Caddy and the log shipper (podman-log-shipper.service) remain separate units.

**Deprecated:** `labour-bureau-services.service` (oneshot) is deprecated; it can show "active" while no containers are running after a reboot. Use `labour-bureau-compose.service` instead.

### Start the stack manually (ad-hoc)

From the `prod` directory (e.g. for testing):

```bash
cd prod
./start-services.sh
# or: podman compose -f docker-compose.prod.yml up -d
```

### Check Service Status

```bash
cd prod && podman compose -f docker-compose.prod.yml ps
# or: podman ps --filter "name=politburo|db|redis|..."
```

### View Logs

```bash
cd prod
# All services
podman compose -f docker-compose.prod.yml logs -f

# Specific service
podman compose -f docker-compose.prod.yml logs -f politburo
```

### Stop the Stack

```bash
cd prod && ./stop-services.sh
# or: podman compose -f docker-compose.prod.yml down
```

### Recovery: all containers are down

If every container is stopped (e.g. after a reboot or manual stop) and you want the stack back up:

**Option 1 – systemd (if you use labour-bureau-compose.service)**

```bash
sudo systemctl start labour-bureau-compose.service
```

**Option 2 – manual start from prod**

```bash
cd ~/projects/labour-bureau/prod
./start-services.sh
```

If you see errors like **"container name already in use"** (exited containers with the same names as the stack), remove those and start again:

```bash
cd ~/projects/labour-bureau/prod
./scripts/clean-exited-for-compose.sh
./start-services.sh
```

Or by hand (only removes exited containers):

```bash
podman rm -f politburo comrade-bot db redis prometheus loki promtail grafana 2>/dev/null || true
podman rm -f prod_db_1 prod_politburo_1 prod_comrade-bot_1 2>/dev/null || true
cd ~/projects/labour-bureau/prod && ./start-services.sh
```

### Stop and Remove Volumes (⚠️ Destroys Data)

```bash
cd prod && podman compose -f docker-compose.prod.yml down -v
```

## Service Access

- **Politburo API**: Accessible via Caddy at `https://comradebot.cc`
- **Grafana**: Accessible via Caddy at `https://monitor.comradebot.cc`
- **Prometheus**: Internal only (port 9090, not exposed)
- **Loki**: Internal only (port 3100, not exposed)
- **Promtail**: Internal only (port 9080, not exposed)

## Environment Variable Organization

Environment variables are organized by service type to prevent unnecessary secret sharing:

- **`politburo.env`**: Politburo application configuration (database, Redis, API keys, security)
- **`comrade-bot.env`**: Comrade-bot configuration (Discord token, API URL)
- **`database.env`**: PostgreSQL database credentials
- **`cache.env`**: Redis cache password
- **`monitoring.env`**: Grafana admin password

Each service only receives the environment variables it needs, improving security isolation.

## Troubleshooting

### Docker Data Root Issues

If Docker volumes aren't being created on the mounted disk:

1. Verify Docker data-root configuration:
   ```bash
   docker info | grep "Docker Root Dir"
   ```

2. Ensure the directory exists and has proper permissions:
   ```bash
   sudo mkdir -p /mnt/HC_Volume_104770220/docker
   sudo chown -R root:root /mnt/HC_Volume_104770220/docker
   ```

3. Restart Docker:
   ```bash
   sudo systemctl restart docker
   ```

### Redis Authentication Errors

Ensure `REDIS_PASSWORD` in `cache.env` matches the password used in `politburo.env`.

### Service Health Checks

Check service health:

```bash
cd prod && podman compose -f docker-compose.prod.yml ps
```

All services should show "healthy" status.

### View Service Logs

```bash
cd prod
podman compose -f docker-compose.prod.yml logs -f politburo
podman compose -f docker-compose.prod.yml logs -f db
podman compose -f docker-compose.prod.yml logs -f
```

## Backup Recommendations

### PostgreSQL Backup

```bash
cd prod && podman compose -f docker-compose.prod.yml exec db pg_dump -U ${POSTGRES_USER} ${POSTGRES_DB} > backup.sql
```

### Redis Backup

Redis data is persisted via AOF (Append Only File) in the `redis-prod` volume.

## Updates

To deploy updated politburo or comrade-bot (rebuild and recreate that service):

```bash
cd prod
./deploy-services.sh politburo    # or comrade-bot, or all
```

To pull images and rebuild the whole stack:

```bash
cd prod
podman compose -f docker-compose.prod.yml pull
podman compose -f docker-compose.prod.yml up -d --build
```
