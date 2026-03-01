# ─── Stage 1: Builder ────────────────────────────────────────────────────────
FROM node:20-slim AS builder
WORKDIR /app

# Install build dependencies for canvas and ts compilation
RUN apt-get update && apt-get install -y \
  python3 \
  build-essential \
  libcairo2-dev \
  libpango1.0-dev \
  libjpeg-dev \
  libgif-dev \
  librsvg2-dev \
  && rm -rf /var/lib/apt/lists/*

COPY package.json package-lock.json ./
RUN npm ci

COPY tsconfig.json ./
COPY src ./src
COPY docs ./docs
RUN npm run build

# ─── Stage 2: Production ─────────────────────────────────────────────────────
FROM node:20-slim AS prod
WORKDIR /app

# Runtime dependencies for canvas
RUN apt-get update && apt-get install -y \
  python3 \
  libcairo2 \
  libpango-1.0-0 \
  libjpeg62-turbo \
  libgif7 \
  librsvg2-2 \
  && rm -rf /var/lib/apt/lists/*

COPY package.json package-lock.json ./
RUN npm ci --omit=dev

COPY --from=builder /app/dist ./dist
COPY --from=builder /app/docs ./docs

# Copy deploy script for command deployment
COPY deploy.sh ./
RUN chmod +x ./deploy.sh

# Optional: run deploy commands
# CMD ["node", "dist/deploy-commands.js"]

# Default command (your bot)
CMD ["node", "dist/index.js"]
