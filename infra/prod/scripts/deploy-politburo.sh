#!/bin/bash
# deploy-politburo.sh - Rebuild and recreate politburo via compose (legacy on-host build).
# New VPS: pull ghcr.io/<owner>/politburo:<sha> and set image in compose or k8s instead.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROD_DIR="${SCRIPT_DIR}/.."
source "${SCRIPT_DIR}/common.sh"

COMPOSE_FILE="${PROD_DIR}/docker-compose.prod.yml"
CLEAN_BUILD=false
if [[ "${1:-}" == "--clean" ]]; then
    CLEAN_BUILD=true
fi

cd "$PROD_DIR"
if [[ "$CLEAN_BUILD" == "true" ]]; then
    podman compose -f "$COMPOSE_FILE" build --no-cache politburo
else
    podman compose -f "$COMPOSE_FILE" build politburo
fi
podman compose -f "$COMPOSE_FILE" up -d politburo

sleep 2
if podman ps --format "{{.Names}}" | grep -q "^politburo$"; then
    echo -e "${GREEN}Politburo is running${NC}"
else
    podman compose -f "$COMPOSE_FILE" logs --tail 50 politburo || true
    exit 1
fi
