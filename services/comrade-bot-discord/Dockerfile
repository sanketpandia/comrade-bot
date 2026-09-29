# syntax=docker/dockerfile:1
# Build from repository root: docker build -f services/comrade-bot-discord/Dockerfile .

FROM node:20-slim AS builder
WORKDIR /repo

RUN apt-get update && apt-get install -y \
  python3 \
  build-essential \
  libcairo2-dev \
  libpango1.0-dev \
  libjpeg-dev \
  libgif-dev \
  librsvg2-dev \
  && rm -rf /var/lib/apt/lists/*

COPY services/comrade-bot-discord/package.json services/comrade-bot-discord/package-lock.json ./services/comrade-bot-discord/
WORKDIR /repo/services/comrade-bot-discord
RUN npm ci

COPY openapi /repo/openapi
COPY services/comrade-bot-discord/tsconfig.json ./
COPY services/comrade-bot-discord/src ./src
COPY services/comrade-bot-discord/assets ./assets
RUN npm run api:generate && npm run build

FROM node:20-slim AS prod
WORKDIR /app

RUN apt-get update && apt-get install -y \
  python3 \
  libcairo2 \
  libpango-1.0-0 \
  libjpeg62-turbo \
  libgif7 \
  librsvg2-2 \
  && rm -rf /var/lib/apt/lists/*

COPY services/comrade-bot-discord/package.json services/comrade-bot-discord/package-lock.json ./
RUN npm ci --omit=dev

COPY --from=builder /repo/services/comrade-bot-discord/dist ./dist
COPY --from=builder /repo/services/comrade-bot-discord/assets ./assets

COPY services/comrade-bot-discord/deploy.sh ./
RUN chmod +x ./deploy.sh

CMD ["node", "dist/index.js"]
