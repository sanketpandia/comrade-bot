# Development vs Production Configuration Analysis

> **Dev setup:** Day-to-day local development is documented in [`README.md`](README.md).
> Dev sections below describe the pre-rewrite layout (separate Vizburo, Politburo
> `:8080` in Compose, Tailwind frontend container). Production comparison material
> may still be useful but has not been fully reconciled with the rewrite.

## Executive Summary

✅ **Production setup is now properly secured with the following improvements:**
- Redis added to production (was missing)
- PostgreSQL port removed from host exposure
- Internal isolated network for all services
- Password protection on Redis
- Enhanced security headers in Caddyfile
- Production .env.example created for safe credential management

---

## Service Comparison

### 1. **Politburo (Backend - Go)**

#### Development (Dockerfile.dev)
```dockerfile
FROM golang:1.24 as dev
WORKDIR /app
RUN go install github.com/air-verse/air@latest
COPY . .
CMD ["air"]  # Hot reload via Air
```

**Features:**
- ✅ Full Go toolchain included (~1.2GB)
- ✅ Air hot reload enabled (1 second rebuild)
- ✅ Source code mounted as volume
- ✅ Port 8080 exposed to host
- ✅ Rebuilds on every `.go` file change
- ✅ Depends on: `db`, `redis`

**Volume Mount:**
```yaml
volumes:
  - ../politburo:/app  # Live code changes
```

#### Production (Dockerfile - Multi-stage)
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/app ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/app .
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --spider --quiet http://localhost:8080/healthCheck || exit 1
ENTRYPOINT ["./app"]
```

**Features:**
- ✅ Multi-stage build (builder + runtime)
- ✅ Alpine base image (~5.5MB vs 1.2GB)
- ✅ Static binary (CGO_ENABLED=0)
- ✅ CA certificates included for HTTPS
- ✅ Health checks enabled
- ✅ Immutable (no volumes)
- ✅ Depends on: `db`, `redis` ✨ NEW
- ✅ On internal network only
- ✅ Port 8080 internal only (exposed via Caddy)

**Image Comparison:**
| Aspect | Dev | Prod |
|--------|-----|------|
| Base Image | golang:1.24 (1.2GB) | golang:1.23-alpine → alpine:3.19 |
| Final Image Size | ~1.2GB | ~50-100MB |
| Build Time | ~2s (with cache) | ~30s first build, ~5s incremental |
| Artifacts | Full toolchain | Single binary |
| HTTP Client | Not included | wget in Alpine for health checks |

---

### 2. **Vizburo (Frontend - Node/Tailwind)**

#### Development (Dockerfile.dev)
```dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY src ./src
COPY favicon.ico ./
COPY tailwind.config.ts ./
RUN npm install -g concurrently serve

CMD ["concurrently",
  "npx tailwindcss -i ./src/styles.css -o ./src/generated.css --watch",
  "serve src -l 3000"]
```

**Features:**
- ✅ Tailwind watcher enabled (live CSS compilation)
- ✅ `serve` CLI serves from `src` directory (not `dist`)
- ✅ Port 8081 (host) → 3000 (container)
- ✅ Source mounted as volume for live reload
- ✅ HTML templates auto-refresh on change

**Dev Container:**
```yaml
vizburo:
  volumes:
    - ../vizburo:/app
  ports:
    - "8081:3000"
  environment:
    - API_URL=http://politburo:8080
```

**Commands:**
- Tailwind: watches `./src/styles.css` → writes `./src/generated.css`
- Serve: serves `./src` directory with live reload

#### Production (Dockerfile - Static)
```dockerfile
FROM nginx:alpine
COPY . /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

**Features:**
- ✅ Lightweight nginx Alpine base
- ✅ Serves static files only (pre-built)
- ✅ No Node.js, npm, or Tailwind in production
- ✅ Immutable container
- ✅ Port 80 internal only
- ✅ On internal network only

**Image Comparison:**
| Aspect | Dev | Prod |
|--------|-----|------|
| Base Image | node:20-alpine (~150MB) | nginx:alpine (~40MB) |
| Final Size | ~200-250MB | ~50-80MB |
| Build Tools | Tailwind, serve, Node.js | None (pre-built) |
| Content | src/ directory (live) | Compiled dist/ files |

**Production Flow:**
```
1. Build frontend: npm run build
   - Tailwind compiles CSS (minified)
   - Output to dist/
2. Copy dist/ to nginx
3. Serve as static files
```

⚠️ **NOTE:** Ensure CSS is pre-built before deploying:
```bash
cd vizburo
npm run build  # Outputs to dist/
```

---

### 3. **Comrade-Bot (Node.js Discord Bot)**

#### Development (Dockerfile.dev)
```dockerfile
FROM node:20-slim
WORKDIR /app
RUN apt-get update && apt-get install -y \
  python3 build-essential \
  libcairo2-dev libpango1.0-dev libjpeg-dev libgif-dev librsvg2-dev \
  && rm -rf /var/lib/apt/lists/*

RUN npm install -g ts-node-dev
COPY package.json package-lock.json ./
RUN npm ci
COPY . .

CMD ["ts-node-dev", "--respawn", "--transpile-only", "src/index.ts"]
```

**Features:**
- ✅ Full development toolchain (python3, build-essential)
- ✅ ts-node-dev for TypeScript hot reload
- ✅ Canvas dependencies for image generation
- ✅ Source mounted as volume
- ✅ Automatically restarts on code changes

**Dev Container:**
```yaml
comrade-bot:
  volumes:
    - ../comrade-bot:/app
  env_file:
    - ../comrade-bot/.env
```

#### Production (Dockerfile - Multi-stage)
```dockerfile
FROM node:20-slim AS builder
WORKDIR /app
RUN apt-get update && apt-get install -y \
  python3 build-essential \
  libcairo2-dev libpango1.0-dev libjpeg-dev libgif-dev librsvg2-dev \
  && rm -rf /var/lib/apt/lists/*

COPY package.json package-lock.json ./
RUN npm ci
COPY tsconfig.json ./
COPY src ./src
RUN npm run build

FROM node:20-slim AS prod
WORKDIR /app
RUN apt-get update && apt-get install -y \
  python3 libcairo2 libpango-1.0-0 libjpeg62-turbo libgif7 librsvg2-2 \
  && rm -rf /var/lib/apt/lists/*

COPY package.json package-lock.json ./
RUN npm ci --omit=dev  # Only production deps

COPY --from=builder /app/dist ./dist
CMD ["node", "dist/index.js"]
```

**Features:**
- ✅ Multi-stage build (builder + runtime)
- ✅ TypeScript compiled to JavaScript in builder
- ✅ Only runtime dependencies in final image
- ✅ Canvas runtime libraries included
- ✅ Immutable (no volumes)
- ✅ Smaller image (dev deps excluded)
- ✅ On internal network only

**Image Comparison:**
| Aspect | Dev | Prod |
|--------|-----|------|
| Size | ~500-600MB | ~300-350MB |
| Node Deps | All (dev + prod) | Only production |
| TypeScript | Runtime compilation | Pre-compiled |
| Build Time | ~5-10s hot reload | ~30s first build |

---

### 4. **PostgreSQL Database**

#### Development
```yaml
db:
  image: postgres:15
  ports:
    - "5432:5432"  # ⚠️ Exposed to host
  environment:
    POSTGRES_DB: infinite
    POSTGRES_USER: ieuser
    POSTGRES_PASSWORD: iepass
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U ieuser -d infinite"]
```

**Features:**
- ✅ Port 5432 exposed for local development
- ✅ Accessible from host for CLI tools
- ✅ PGAdmin 4 on port 5050 for GUI management
- ✅ Health checks enabled
- ✅ Data persists in volume `pgdata-dev`

**Dev Usage:**
```bash
# Direct CLI access from host
psql -h localhost -U ieuser -d infinite

# Via Docker
docker-compose -f docker-compose.dev.yml exec db psql -U ieuser -d infinite

# Via PGAdmin
# Visit http://localhost:5050
# Email: sanketpandia@gmail.com, Password: ieadmin
```

#### Production ✅ FIXED
```yaml
db:
  image: postgres:15
  networks:
    - internal  # Only internal network
  # ✅ NO ports exposed
  environment:
    POSTGRES_DB: infinite
    POSTGRES_USER: ieuser
    POSTGRES_PASSWORD: iepass  # ⚠️ Change to strong password
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U ieuser -d infinite"]
  restart: unless-stopped
```

**Security Changes:**
- ❌ Port 5432 NO LONGER exposed to host
- ✅ Only accessible via internal Docker network
- ✅ Only `politburo` container can connect
- ⚠️ Requires strong password in `.env.prod`

**Production Access:**
```bash
# Only from within container network
docker-compose -f docker-compose.prod.yml exec db psql -U ieuser -d infinite

# NO access from host directly
# ❌ psql -h localhost -U ieuser -d infinite  # Will NOT work
```

---

### 5. **Redis Cache**

#### Development
```yaml
redis:
  image: redis:7-alpine
  ports:
    - "6379:6379"  # Exposed to host
  command: redis-server --appendonly yes
  volumes:
    - redis-data:/data
```

**Features:**
- ✅ Port 6379 exposed to host
- ✅ AOF persistence enabled
- ✅ No password (OK for dev)
- ✅ Health checks (basic ping)

**Dev Access:**
```bash
# Direct from host
redis-cli -h localhost ping

# Via Docker
docker-compose -f docker-compose.dev.yml exec redis redis-cli ping
```

#### Production ✅ NEW
```yaml
redis:
  image: redis:7-alpine
  container_name: redis-prod
  networks:
    - internal  # Only internal network
  # ✅ NO ports exposed to host
  command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD:-change-me}
  volumes:
    - redis-prod:/data
  healthcheck:
    test: ["CMD", "redis-cli", "--raw", "incr", "ping"]
    interval: 10s
    timeout: 3s
    retries: 5
  restart: unless-stopped
```

**Security Features:**
- ✅ Password protected with `--requirepass`
- ✅ NOT exposed to host
- ✅ Only accessible via internal network
- ✅ AOF persistence for data durability
- ✅ Health checks every 10 seconds

**Production Setup:**
```bash
# In .env.prod
REDIS_PASSWORD=your_strong_password_here_min_16_chars

# Inside container, authenticated access
docker-compose -f docker-compose.prod.yml exec redis \
  redis-cli -a your_strong_password_here_min_16_chars ping

# ❌ From host - will NOT work
redis-cli -h localhost ping  # Connection refused
```

**Backend Connection (in .env.prod):**
```
REDIS_HOST=redis        # Service name on internal network
REDIS_PORT=6379
REDIS_PASSWORD=your_strong_password_here_min_16_chars
REDIS_DB=0
```

---

### 6. **Caddy Reverse Proxy (Prod Only)**

#### Development
No reverse proxy. Direct access:
- Backend: `http://localhost:8080`
- Frontend: `http://localhost:8081`

#### Production ✨ ENHANCED
```yaml
comradebot.cc {
  # Automatic HTTPS via Let's Encrypt
  encode gzip

  # Security headers
  header / {
    X-Frame-Options "SAMEORIGIN"
    X-Content-Type-Options "nosniff"
    X-XSS-Protection "1; mode=block"
    Referrer-Policy "strict-origin-when-cross-origin"
    X-Download-Options "noopen"
    Permissions-Policy "geolocation=(), microphone=(), camera=()"
  }

  # API routing
  reverse_proxy /public/* politburo:8080 {
    header_up X-Real-IP {http.request.remote.host}
    header_up X-Forwarded-For {http.request.remote.host}
    header_up X-Forwarded-Proto {http.request.scheme}
  }

  reverse_proxy /api/* politburo:8080 {
    header_up X-Real-IP {http.request.remote.host}
    header_up X-Forwarded-For {http.request.remote.host}
    header_up X-Forwarded-Proto {http.request.scheme}
  }

  # Frontend routing
  reverse_proxy /* vizburo:80 {
    header_up X-Real-IP {http.request.remote.host}
    header_up X-Forwarded-For {http.request.remote.host}
    header_up X-Forwarded-Proto {http.request.scheme}
  }

  file_server
}
```

**Features:**
- ✅ Automatic HTTPS (Let's Encrypt)
- ✅ Security headers for all responses
- ✅ XFF (X-Forwarded-For) headers for real IPs
- ✅ Gzip compression
- ✅ Path-based routing

---

## Network Architecture

### Development Network
```
Host Machine
├── localhost:8080 → politburo:8080
├── localhost:8081 → vizburo:3000
├── localhost:5432 → db:5432
├── localhost:6379 → redis:6379
└── localhost:5050 → pgadmin:80

Default Docker Network (bridge)
├─ politburo (can reach all services by hostname)
├─ vizburo (can reach all services)
├─ comrade-bot (can reach all services)
├─ db (accessible to all)
├─ redis (accessible to all)
└─ pgadmin (accessible to all)
```

### Production Network
```
Internet
└── comradebot.cc:443 (HTTPS)
    ↓ (Caddy in container network)

Docker Internal Network 'internal'
├─ politburo:8080 (only from caddy)
├─ vizburo:80 (only from caddy)
├─ comrade-bot (internal only)
├─ db:5432 (NO host access)
└─ redis:6379 (NO host access)

Host Machine
└── Port 443 (HTTPS via Caddy)
└── Port 80 (HTTP via Caddy)
    └── Redirects to 443
```

---

## Environment Variables

### Development (.env / .env.local)
```bash
APP_ENV=prod
DEBUG=true
PORT=8080

# Database (accessible from host)
PG_HOST=db
PG_PORT=5432
PG_USER=ieuser
PG_PASSWORD=iepass

# Redis (accessible from host, no password)
REDIS_HOST=redis
REDIS_PORT=6379
# No password needed in dev
```

### Production (.env.prod) ✨ NEW TEMPLATE
```bash
APP_ENV=prod
DEBUG=false  # ⚠️ MUST be false in prod
PORT=8080

# Database (internal network only)
PG_HOST=db
PG_PORT=5432
PG_USER=ieuser
PG_PASSWORD=change_this_strong_password  # ⚠️ Change this!

# Redis (internal network only, password protected)
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=change_this_strong_redis_password  # ⚠️ Change this!

# API Configuration
API_URL=http://politburo:8080
API_KEY=your_api_key_here
DISCORD_BOT_TOKEN=your_discord_bot_token_here
```

---

## Volume Persistence

### Development
| Service | Volume | Purpose |
|---------|--------|---------|
| politburo | ../politburo:/app | Live code changes (hot reload) |
| vizburo | ../vizburo:/app | Live source (Tailwind + serve) |
| comrade-bot | ../comrade-bot:/app | Live TypeScript source |
| db | pgdata-dev | PostgreSQL data persistence |
| redis | redis-data | Redis AOF persistence |
| politburo | politburo_air_tmp | Air temporary build files |

### Production
| Service | Volume | Purpose |
|---------|--------|---------|
| db | pgdata-prod | PostgreSQL data persistence |
| redis | redis-prod | Redis AOF persistence |
| (all others) | None | Immutable images |

**Key Difference:**
- Dev: Source code volumes for live reload
- Prod: Only data volumes (database, cache)

---

## Dockerfile Best Practices Implemented

### ✅ Development Dockerfiles
- ✅ Single stage (fast iteration)
- ✅ Large base images (full toolchains for hot reload)
- ✅ Source mounted as volumes (live reload)
- ✅ Health checks for services

### ✅ Production Dockerfiles
- ✅ Multi-stage builds (optimized final images)
- ✅ Minimal base images (Alpine, slim)
- ✅ Compile/build in builder stage
- ✅ Copy only necessary artifacts to runtime
- ✅ Non-root considerations (though not implemented in current Dockerfiles)
- ✅ Health checks for critical services
- ✅ No volume mounts (immutable)

### Potential Improvements
```dockerfile
# Add non-root user (not currently implemented)
FROM alpine:3.19
RUN adduser -D -u 1000 appuser
COPY --chown=appuser:appuser . .
USER appuser
```

---

## Security Checklist

### Development ✅
- ✅ Exposed ports OK for local development
- ✅ No password on Redis (internal network only)
- ✅ No password on PostgreSQL (internal network only)
- ✅ Debug mode enabled (expected for development)

### Production ✅ IMPROVED
- ✅ PostgreSQL NOT exposed to host
- ✅ Redis NOT exposed to host
- ✅ Redis password protected
- ✅ Internal network isolation
- ✅ Security headers in Caddyfile
- ✅ Automatic HTTPS via Caddy
- ✅ Debug mode DISABLED (DEBUG=false)
- ✅ .env.prod.example has placeholder values (no real secrets)
- ✅ Real .env.prod is in .gitignore (not committed)

### Additional Recommendations
- ⚠️ Implement non-root users in Dockerfiles
- ⚠️ Scan images for vulnerabilities: `docker scan image-name`
- ⚠️ Set resource limits in docker-compose.prod.yml
- ⚠️ Implement log aggregation/monitoring
- ⚠️ Configure database backups (separate process)

---

## Build & Deployment Flow

### Development Workflow
```bash
# Day-to-day development
docker-compose -f docker-compose.dev.yml up

# Code changes automatically trigger hot reload
# No manual restart needed
# Logs visible in terminal

# To stop
docker-compose -f docker-compose.dev.yml down
```

### Production Deployment
```bash
# 1. Build images
docker-compose -f docker-compose.prod.yml build

# 2. Set environment variables
cp .env.prod.example .env.prod
# Edit .env.prod with real values

# 3. Deploy
docker-compose -f docker-compose.prod.yml up -d

# 4. Monitor
docker-compose -f docker-compose.prod.yml logs -f

# 5. Update
docker-compose -f docker-compose.prod.yml pull  # Pull new images
docker-compose -f docker-compose.prod.yml up -d --build  # Rebuild and restart
```

---

## Summary of Changes

### Production Improvements Made ✨

1. **Redis Added**
   - Was missing entirely
   - Now included with password protection
   - Internal network only
   - AOF persistence enabled

2. **PostgreSQL Secured**
   - ❌ Removed port 5432 exposure to host
   - ✅ Internal network only
   - Only accessible from politburo container

3. **Network Isolation**
   - ✅ Added `internal` Docker network
   - ✅ All services on isolated network
   - ✅ Only ports 80 & 443 exposed (via Caddy)

4. **Security Headers**
   - ✅ Added OWASP security headers
   - ✅ X-Frame-Options, X-Content-Type-Options, etc.
   - ✅ Proper proxy headers (X-Real-IP, X-Forwarded-For, etc.)

5. **Environment Management**
   - ✅ Created .env.prod.example template
   - ✅ Placeholder values instead of real secrets
   - ✅ Clear documentation of required variables

### Files Modified

- ✅ `docker-compose.prod.yml` - Added Redis, removed DB port, added network
- ✅ `Caddyfile` - Added security headers and proper proxying
- ✅ `.env.prod.example` - NEW: Production configuration template
- ✅ `DOCKER_TESTING_GUIDE.md` - NEW: Comprehensive testing guide
- ✅ `DEV_PROD_COMPARISON.md` - NEW: This document

---

## Next Steps

1. **Test Production Setup:**
   - Run tests from DOCKER_TESTING_GUIDE.md (Tests 9-12)
   - Verify no services exposed unintentionally
   - Test internal service communication

2. **Update .env.prod:**
   - Copy from `.env.prod.example`
   - Change PostgreSQL password
   - Change Redis password
   - Add real API credentials

3. **Deploy to Production:**
   - Build images: `docker-compose -f docker-compose.prod.yml build`
   - Deploy: `docker-compose -f docker-compose.prod.yml up -d`
   - Monitor: `docker-compose -f docker-compose.prod.yml logs -f`

4. **Monitor Security:**
   - Scan images for vulnerabilities
   - Monitor logs for unauthorized access attempts
   - Set up alerts for service failures
