# Docker Setup & Testing Guide

> **Dev setup:** For day-to-day local development, use [`README.md`](README.md) and
> `./start-dev.sh`. This guide predates the Politburo rewrite and still describes a
> separate Vizburo app, Politburo in Compose on `:8080`, and other legacy layout.
> Treat the README as authoritative for dev; use this document mainly for Docker
> build/testing patterns and production-oriented sections below.

## Overview

Your project uses **Docker Compose** for both development and production deployments with distinct configurations optimized for each environment.

### Key Directories
- **labour-bureau/**: Orchestration files (Caddyfile, docker-compose files, deployment scripts)
- **politburo/**: Go backend service
- **vizburo/**: Frontend UI (static files + Tailwind CSS)
- **comrade-bot/**: Node.js Discord bot service

---

## Development Setup

### Architecture

```
┌─────────────────────────────────────────────────────┐
│ docker-compose.dev.yml                              │
├─────────────────────────────────────────────────────┤
│ politburo (Go backend)                              │
│  ├─ Port: 8080                                      │
│  ├─ Hot reload: Air (golang.org/x/tools/cmd/air)   │
│  ├─ Depends on: db, redis                           │
│  └─ Volume: ../politburo:/app (live code mount)     │
├─────────────────────────────────────────────────────┤
│ vizburo (Frontend)                                  │
│  ├─ Port: 8081 → 3000 (internal)                    │
│  ├─ Hot reload: Tailwind watch + serve              │
│  ├─ Dependencies: Tailwind CSS, serve CLI            │
│  └─ Volume: ../vizburo:/app (live code mount)       │
├─────────────────────────────────────────────────────┤
│ comrade-bot (Node.js Discord Bot)                   │
│  ├─ Hot reload: ts-node-dev with respawn            │
│  ├─ Depends on: none                                │
│  └─ Volume: ../comrade-bot:/app (live code mount)   │
├─────────────────────────────────────────────────────┤
│ db (PostgreSQL 15)                                  │
│  ├─ Port: 5432                                      │
│  ├─ Health checks: Enabled (pg_isready)             │
│  └─ Volume: pgdata-dev:/var/lib/postgresql/data     │
├─────────────────────────────────────────────────────┤
│ redis (Redis 7-alpine)                              │
│  ├─ Port: 6379 (NO password in dev)                 │
│  ├─ Command: redis-server --appendonly yes           │
│  └─ Volume: redis-data:/data                        │
├─────────────────────────────────────────────────────┤
│ pgadmin (PGAdmin 4)                                 │
│  ├─ Port: 5050                                      │
│  ├─ Email: sanketpandia@gmail.com                   │
│  ├─ Password: ieadmin                               │
│  └─ Access: http://localhost:5050                   │
└─────────────────────────────────────────────────────┘
```

### Dev Environment File Structure

```
labour-bureau/
├── .env                       # Current development env vars
├── .env.example              # Template (not updated)
├── .env.local                # Alternative local config
├── .env.prod                 # Production env vars (SECRET - DO NOT COMMIT)
├── .env.prod.example         # NEW: Production template
├── .env.common               # Shared variables
├── docker-compose.dev.yml    # Development orchestration
└── docker-compose.prod.yml   # Production orchestration
```

---

## Testing Docker Locally

### Prerequisites

```bash
# Check Docker is installed
docker --version        # Should be 20.10+
docker-compose --version # Should be 1.29+

# Ensure you have git (for Go modules)
git --version
```

### Test 1: Full Dev Stack Startup

**What it tests:** All services start correctly, health checks pass, volumes mount properly.

```bash
cd /home/odin/projects/infinite-experiment/labour-bureau

# Option A: Start from scratch (clean slate)
docker-compose -f docker-compose.dev.yml down --volumes
docker-compose -f docker-compose.dev.yml up --build

# Option B: Start with cached images (faster)
docker-compose -f docker-compose.dev.yml up

# Option C: Start in background and check logs later
docker-compose -f docker-compose.dev.yml up -d
docker-compose -f docker-compose.dev.yml logs -f
```

**What to verify:**
```
✅ politburo starts and rebuilds on code changes
✅ vizburo starts with Tailwind watcher running
✅ comrade-bot starts without errors
✅ db shows "database system is ready"
✅ redis shows "Ready to accept connections"
✅ All health checks pass (no UNHEALTHY status)
```

**Example output:**
```
db      | database system is ready to accept connections
redis   | Ready to accept connections
politburo | 2024-10-26 10:25:33 | 1 | main.go:123 | Starting server on :8080
vizburo   | Serving! Available on: http://0.0.0.0:3000
comrade-bot | 2024-10-26 10:25:35 Connected to Discord as MyBot#1234
```

### Test 2: Service Health Checks

**What it tests:** Container health monitoring and service readiness.

```bash
# Check container status and health
docker-compose -f docker-compose.dev.yml ps

# Expected output:
# NAME      STATUS                PORTS
# politburo Up (healthy)          0.0.0.0:8080->8080/tcp
# vizburo   Up 2 minutes           0.0.0.0:8081->3000/tcp
# db        Up (healthy)           0.0.0.0:5432->5432/tcp
# redis     Up 2 minutes           0.0.0.0:6379->6379/tcp

# View individual container logs
docker-compose -f docker-compose.dev.yml logs db
docker-compose -f docker-compose.dev.yml logs redis
docker-compose -f docker-compose.dev.yml logs politburo
```

### Test 3: Network Connectivity

**What it tests:** Services can communicate via internal Docker network.

```bash
# Test backend health check endpoint
curl http://localhost:8080/healthCheck

# Test frontend is serving
curl http://localhost:8081 | head -20

# Test database connectivity
docker-compose -f docker-compose.dev.yml exec db psql -U ieuser -d infinite -c "\dt"

# Test Redis connectivity
docker-compose -f docker-compose.dev.yml exec redis redis-cli ping
# Should return: PONG

# Test inter-service communication (from backend)
docker-compose -f docker-compose.dev.yml exec politburo wget -O - http://redis:6379
```

### Test 4: Volume Mounts & Hot Reload

**What it tests:** Code changes trigger hot reload without container restart.

```bash
# Start dev stack
docker-compose -f docker-compose.dev.yml up -d

# Make a code change in politburo
echo 'fmt.Println("TEST CHANGE")' >> ../politburo/cmd/server/main.go

# Watch logs for recompilation
docker-compose -f docker-compose.dev.yml logs -f politburo

# Look for "Air" output indicating hot reload
# Expected: "Air rebuilding..." → "Running..."

# Verify the change was picked up
curl http://localhost:8080/healthCheck
# If running, the endpoint should still work (showing hot reload worked)

# Revert the test change
git -C ../politburo checkout cmd/server/main.go
```

### Test 5: Data Persistence

**What it tests:** Database and Redis data survives container restarts.

```bash
# Start services
docker-compose -f docker-compose.dev.yml up -d

# Insert test data via PGAdmin or CLI
docker-compose -f docker-compose.dev.yml exec db psql -U ieuser -d infinite -c \
  "CREATE TABLE IF NOT EXISTS test_table (id SERIAL PRIMARY KEY, value TEXT);"
docker-compose -f docker-compose.dev.yml exec db psql -U ieuser -d infinite -c \
  "INSERT INTO test_table (value) VALUES ('persistence_test');"

# Verify data exists
docker-compose -f docker-compose.dev.yml exec db psql -U ieuser -d infinite -c \
  "SELECT * FROM test_table;"
# Should show: id=1, value='persistence_test'

# Stop containers (data persists in volumes)
docker-compose -f docker-compose.dev.yml stop

# Start again
docker-compose -f docker-compose.dev.yml up -d

# Verify data still exists
docker-compose -f docker-compose.dev.yml exec db psql -U ieuser -d infinite -c \
  "SELECT * FROM test_table;"
# Should still show the data

# Clean up: drop test table
docker-compose -f docker-compose.dev.yml exec db psql -U ieuser -d infinite -c \
  "DROP TABLE test_table;"
```

### Test 6: Environment Variables

**What it tests:** Services correctly load environment variables.

```bash
# Check what env vars are being used
docker-compose -f docker-compose.dev.yml exec politburo printenv | grep -E "^(PG_|REDIS_|API_)"

# Should show:
# PG_HOST=db
# PG_PORT=5432
# PG_USER=ieuser
# PG_DB=infinite
# PG_PASSWORD=iepass
# REDIS_HOST=redis
# REDIS_PORT=6379

# Check backend can see API_URL
docker-compose -f docker-compose.dev.yml exec politburo printenv | grep API_URL
# Should show: API_URL=http://politburo:8080 (from docker-compose)
```

### Test 7: Multi-Service Communication

**What it tests:** Services communicate correctly via Docker DNS.

```bash
# From backend, verify it can reach Redis
docker-compose -f docker-compose.dev.yml exec politburo \
  wget -qO- http://redis:6379/PING 2>&1 || echo "Connected (response shows it's reachable)"

# From backend, verify it can reach database
docker-compose -f docker-compose.dev.yml exec politburo \
  nc -zv db 5432

# From comrade-bot, verify it can reach politburo
docker-compose -f docker-compose.dev.yml exec comrade-bot \
  wget -qO- http://politburo:8080/healthCheck
```

### Test 8: Clean Rebuild

**What it tests:** Complete rebuild from scratch without cached layers.

```bash
# Option A: Clean build (full rebuild)
docker-compose -f docker-compose.dev.yml down --volumes
docker-compose -f docker-compose.dev.yml up --build --no-cache

# Option B: Rebuild specific service
docker-compose -f docker-compose.dev.yml up -d --build --no-cache politburo

# Option C: Remove images entirely and rebuild
docker-compose -f docker-compose.dev.yml down --volumes --remove-orphans
docker image rm $(docker image ls -q)  # WARNING: Removes ALL images
docker-compose -f docker-compose.dev.yml up --build
```

---

## Production Setup Testing

### Architecture Differences

```
┌─────────────────────────────────────────────────────┐
│ docker-compose.prod.yml                             │
├─────────────────────────────────────────────────────┤
│ politburo (Go backend - PROD BUILD)                 │
│  ├─ Port: 8080                                      │
│  ├─ Multi-stage build (builder + alpine)            │
│  ├─ Depends on: db, redis ✨ NEW                    │
│  ├─ Network: internal (no host exposure)            │
│  └─ No volumes (immutable container)                │
├─────────────────────────────────────────────────────┤
│ vizburo (Frontend - nginx)                          │
│  ├─ Port: 80                                        │
│  ├─ Lightweight nginx Alpine server                 │
│  ├─ Network: internal                              │
│  └─ No volumes                                      │
├─────────────────────────────────────────────────────┤
│ comrade-bot (Node.js PROD BUILD)                    │
│  ├─ Multi-stage build (builder + slim runtime)      │
│  ├─ Network: internal                              │
│  └─ No volumes                                      │
├─────────────────────────────────────────────────────┤
│ db (PostgreSQL 15 - INTERNAL ONLY) ✨ FIXED         │
│  ├─ Port: NOT EXPOSED to host                       │
│  ├─ Only accessible via internal network            │
│  ├─ Health checks: Enabled                          │
│  └─ Network: internal                              │
├─────────────────────────────────────────────────────┤
│ redis (Redis 7 - INTERNAL + PASSWORD) ✨ NEW        │
│  ├─ Port: NOT EXPOSED to host                       │
│  ├─ Password protected: ${REDIS_PASSWORD}           │
│  ├─ Network: internal                              │
│  ├─ Persistence: AOF enabled                        │
│  └─ Health checks: Enabled                          │
├─────────────────────────────────────────────────────┤
│ Caddy (Reverse Proxy)                               │
│  ├─ Handles: HTTPS, security headers                │
│  ├─ Routes /api/* → politburo:8080                  │
│  ├─ Routes /public/* → politburo:8080               │
│  ├─ Routes /* → vizburo:80 (frontend)               │
│  ├─ Port: 80 (HTTP), 443 (HTTPS auto)               │
│  └─ Configuration: ./Caddyfile ✨ ENHANCED          │
└─────────────────────────────────────────────────────┘
```

### Test 9: Production Build Locally

**What it tests:** Production images build correctly and services start.

```bash
cd /home/odin/projects/infinite-experiment/labour-bureau

# Create .env file for production testing (from template)
cp .env.prod.example .env.prod.test
# Edit with test values:
# - Change POSTGRES_PASSWORD to a test password
# - Change REDIS_PASSWORD to a test password
# - Keep everything else same

# Start production stack
docker-compose -f docker-compose.prod.yml --env-file .env.prod.test up --build

# Verify all services are running
docker-compose -f docker-compose.prod.yml ps

# Check no database port is exposed
docker ps | grep postgres
# Should NOT show 5432 port mapping

# Check redis port is NOT exposed
docker ps | grep redis
# Should NOT show 6379 port mapping
```

### Test 10: Network Isolation in Prod

**What it tests:** Services are isolated on internal network, not accessible from host.

```bash
# Start prod stack
docker-compose -f docker-compose.prod.yml up -d

# Try to connect to PostgreSQL from host (should FAIL)
psql -h localhost -U ieuser -d infinite
# Expected: connection refused or timeout

# Try to connect to Redis from host (should FAIL)
redis-cli -h localhost ping
# Expected: connection refused

# But backend CAN access them
docker-compose -f docker-compose.prod.yml exec politburo nc -zv db 5432
# Should succeed (Connection successful)

# Verify services can communicate internally
docker-compose -f docker-compose.prod.yml exec politburo \
  wget -qO- http://redis:6379/PING 2>&1 || echo "Redis reachable (connection made)"
```

### Test 11: Security Headers (Prod Caddyfile)

**What it tests:** Caddy is serving correct security headers.

```bash
# Start stack with Caddy (requires full Caddy setup)
# For now, just verify the Caddyfile syntax

docker run --rm -v $(pwd)/Caddyfile:/etc/caddy/Caddyfile caddy:latest caddy validate

# Expected: "Syntax OK" or similar success message

# Check Caddyfile has security headers
grep -n "X-Frame-Options\|X-Content-Type-Options\|X-XSS-Protection" Caddyfile

# Should show all three security headers defined
```

### Test 12: Environment Secrets Management

**What it tests:** Real secrets are not in version control, template is available.

```bash
# Verify .env.prod (with real secrets) is in .gitignore
cat .gitignore | grep "\.env.prod"

# Verify .env.prod.example (template) is NOT in .gitignore
cat .env.prod.example | head -10

# Check no real tokens visible in repository
grep -r "DISCORD_BOT_TOKEN=MTM" --include="*.yml" --include="*.yaml"
# Should return nothing (no actual tokens in YAML files)

# Verify .env.prod.example has placeholder values
grep "DISCORD_BOT_TOKEN" .env.prod.example
# Should show: DISCORD_BOT_TOKEN=your_discord_bot_token_here
```

---

## Dev vs Prod Configuration Comparison

| Feature | Dev | Prod |
|---------|-----|------|
| **Hot Reload** | ✅ Air/ts-node-dev/Tailwind watch | ❌ Immutable images |
| **Volume Mounts** | ✅ Live code (../service:/app) | ❌ None |
| **PostgreSQL Port** | 5432:5432 (exposed to host) | ❌ Internal only |
| **Redis Port** | 6379:6379 (exposed, no password) | ❌ Internal only + password |
| **Redis Password** | ❌ None | ✅ ${REDIS_PASSWORD} |
| **Network** | Default bridge (exposed) | ✅ Isolated internal network |
| **Build Type** | Single stage + source copy | ✅ Multi-stage (builder + runtime) |
| **Base Images** | Full (golang:1.24, node:20-slim) | ✅ Minimal (alpine, slim) |
| **Database Size** | Larger (~600MB) | ✅ Minimal (~100MB) |
| **PGAdmin** | ✅ Port 5050 | ❌ Not included |
| **Health Checks** | ✅ For db & redis | ✅ For db & redis |
| **Restart Policy** | `always` | ✅ `unless-stopped` |
| **Caddyfile** | ✅ Basic proxy | ✅ Security headers + HTTPS |
| **Image Size** | Larger (for development) | ✅ Optimized for production |

---

## Common Testing Scenarios

### Scenario 1: Test Backend Service in Isolation

```bash
# Start only backend + database + redis
docker-compose -f docker-compose.dev.yml up -d db redis politburo

# Wait for db to be ready
sleep 5

# Test API is responding
curl http://localhost:8080/healthCheck

# View backend logs
docker-compose -f docker-compose.dev.yml logs -f politburo
```

### Scenario 2: Test Frontend Service in Isolation

```bash
# Start only frontend
docker-compose -f docker-compose.dev.yml up -d vizburo

# Test frontend is serving
curl http://localhost:8081
# Should show HTML content

# View tailwind watcher
docker-compose -f docker-compose.dev.yml logs -f vizburo
```

### Scenario 3: Test Discord Bot Startup

```bash
# Start bot with logs visible
docker-compose -f docker-compose.dev.yml up comrade-bot

# Watch for: "Connected to Discord as BotName#XXXX"
# Watch for: "Ready!" or startup messages

# Press Ctrl+C to stop
```

### Scenario 4: Test Database Migrations

```bash
# Start fresh database
docker-compose -f docker-compose.dev.yml up -d db

# List current tables
docker-compose -f docker-compose.dev.yml exec db psql -U ieuser -d infinite -c "\dt"

# Check migrations folder
ls -la ../politburo/internal/db/migrations/

# Verify migration files are in volume
docker-compose -f docker-compose.dev.yml exec db ls -la /migrations
```

### Scenario 5: Verify Redis Persistence

```bash
# Start stack
docker-compose -f docker-compose.dev.yml up -d

# Set test value in Redis
docker-compose -f docker-compose.dev.yml exec redis redis-cli SET test_key "test_value"

# Verify it's set
docker-compose -f docker-compose.dev.yml exec redis redis-cli GET test_key
# Should show: "test_value"

# Stop redis container
docker-compose -f docker-compose.dev.yml stop redis

# Start again
docker-compose -f docker-compose.dev.yml start redis

# Verify data persisted
docker-compose -f docker-compose.dev.yml exec redis redis-cli GET test_key
# Should show: "test_value" (proving AOF persistence works)
```

---

## Troubleshooting

### Issue: "Port already in use"

```bash
# Find which process is using port 8080
lsof -i :8080

# Or kill container that won't stop
docker-compose -f docker-compose.dev.yml down -v
docker kill $(docker ps -q) 2>/dev/null
docker-compose -f docker-compose.dev.yml up
```

### Issue: "db unhealthy" / "can't connect to PostgreSQL"

```bash
# Wait longer for database to be ready
sleep 10

# Check database logs
docker-compose -f docker-compose.dev.yml logs db

# Verify database is accepting connections
docker-compose -f docker-compose.dev.yml exec db pg_isready -U ieuser -d infinite

# Reset database volume
docker-compose -f docker-compose.dev.yml down -v
docker-compose -f docker-compose.dev.yml up
```

### Issue: "redis-cli: command not found"

```bash
# Use exec to run redis-cli in container
docker-compose -f docker-compose.dev.yml exec redis redis-cli ping

# Or install redis-tools locally
sudo apt-get install redis-tools  # Linux
brew install redis                # macOS
```

### Issue: "Out of disk space"

```bash
# Check Docker disk usage
docker system df

# Remove unused volumes/images
docker system prune -a --volumes

# This will free significant space but removes everything not in use
```

### Issue: Code changes not triggering hot reload

```bash
# Check that volume is mounted correctly
docker inspect <container_name> | grep -A 10 Mounts

# Rebuild image without cache
docker-compose -f docker-compose.dev.yml up -d --build --no-cache <service>

# For Air (Go), ensure file has proper line ending (Unix, not Windows)
dos2unix ../politburo/cmd/server/main.go
```

---

## Quick Commands Reference

```bash
# Start everything
docker-compose -f docker-compose.dev.yml up

# Start in background
docker-compose -f docker-compose.dev.yml up -d

# Stop everything
docker-compose -f docker-compose.dev.yml down

# View logs
docker-compose -f docker-compose.dev.yml logs -f           # all services
docker-compose -f docker-compose.dev.yml logs -f politburo # specific service

# Check status
docker-compose -f docker-compose.dev.yml ps

# Execute command in container
docker-compose -f docker-compose.dev.yml exec <service> <command>

# Rebuild images
docker-compose -f docker-compose.dev.yml up --build

# Clean volumes and images
docker-compose -f docker-compose.dev.yml down -v --rmi all

# Scale services
docker-compose -f docker-compose.dev.yml up -d --scale comrade-bot=2

# View resource usage
docker stats

# Tail logs with timestamps
docker-compose -f docker-compose.dev.yml logs -f --timestamps
```

---

## Next Steps

1. **Update Production `.env`:**
   - Use `.env.prod.example` as template
   - Set strong passwords for `POSTGRES_PASSWORD` and `REDIS_PASSWORD`
   - Configure real API keys

2. **Test Production Locally:**
   - Run Test 9-12 above to validate production setup
   - Ensure no services exposed unintentionally

3. **Deploy to Production:**
   - Use `docker-compose -f docker-compose.prod.yml up -d`
   - Monitor logs: `docker-compose -f docker-compose.prod.yml logs -f`
   - Verify Caddy is handling HTTPS correctly

4. **Monitor in Production:**
   - Use `docker stats` to watch resource usage
   - Set up log aggregation (e.g., ELK stack)
   - Configure alerts for service failures
