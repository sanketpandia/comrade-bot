#!/bin/bash
# deploy-jobhunt.sh - Deploy Jobhunt via podman compose (rebuild and recreate)
# Usage: ./deploy-jobhunt.sh [--clean]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROD_DIR="${SCRIPT_DIR}/.."
source "${SCRIPT_DIR}/common.sh"

COMPOSE_FILE="${PROD_DIR}/docker-compose.prod.yml"

# Check for --clean flag
CLEAN_BUILD=false
if [[ "$1" == "--clean" ]]; then
    CLEAN_BUILD=true
    echo -e "${YELLOW}⚠️  Clean build requested - build without cache${NC}"
fi

echo -e "${GREEN}🚀 Deploying Jobhunt...${NC}"

cd "$PROD_DIR"
if [[ "$CLEAN_BUILD" == "true" ]]; then
    podman compose -f "$COMPOSE_FILE" build --no-cache jobhunt
else
    podman compose -f "$COMPOSE_FILE" build jobhunt
fi
podman compose -f "$COMPOSE_FILE" up -d jobhunt

# Wait a moment for container to start
sleep 2

# Verify
if podman ps --format "{{.Names}}" | grep -q "^jobhunt$"; then
    echo -e "${GREEN}✅ Container is running${NC}"
else
    echo -e "${RED}❌ Container failed to start${NC}"
    podman compose -f "$COMPOSE_FILE" logs --tail 50 jobhunt || true
    exit 1
fi

echo -e "${GREEN}✅ Jobhunt deployed successfully!${NC}"
echo -e "${YELLOW}💡 App available at http://localhost:3001 (Caddy proxies from https://jobs.comradebot.cc)${NC}"
