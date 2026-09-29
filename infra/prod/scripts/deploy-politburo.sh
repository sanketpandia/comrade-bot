#!/bin/bash
# deploy-politburo.sh - Deploy Politburo via podman compose (rebuild and recreate)
# Usage: ./deploy-politburo.sh [--clean]

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

echo -e "${GREEN}🚀 Deploying Politburo...${NC}"

cd "$PROD_DIR"
if [[ "$CLEAN_BUILD" == "true" ]]; then
    podman compose -f "$COMPOSE_FILE" build --no-cache politburo
else
    podman compose -f "$COMPOSE_FILE" build politburo
fi
podman compose -f "$COMPOSE_FILE" up -d politburo

# Wait a moment for container to start
sleep 2

# Verify
if podman ps --format "{{.Names}}" | grep -q "^politburo$"; then
    echo -e "${GREEN}✅ Container is running${NC}"
    if podman exec politburo test -f /app/static/js/flight-map.mjs 2>/dev/null; then
        echo -e "${GREEN}✅ flight-map.mjs found in container${NC}"
    else
        echo -e "${RED}❌ flight-map.mjs NOT found in container${NC}"
    fi
    if podman exec politburo test -f /app/templates/pages/live.html 2>/dev/null; then
        echo -e "${GREEN}✅ live.html template found in container${NC}"
    else
        echo -e "${RED}❌ live.html template NOT found in container${NC}"
    fi
else
    echo -e "${RED}❌ Container failed to start${NC}"
    podman compose -f "$COMPOSE_FILE" logs --tail 50 politburo || true
    exit 1
fi

echo -e "${GREEN}✅ Politburo deployed successfully!${NC}"
echo -e "${YELLOW}💡 Tip: If you see map issues, try hard refresh (Ctrl+Shift+R) or check static files at http://localhost:8080/static/js/flight-map.mjs${NC}"
