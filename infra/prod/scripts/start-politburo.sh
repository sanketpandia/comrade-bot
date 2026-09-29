#!/bin/bash
# start-politburo.sh - Start Politburo service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ENV_DIR="${SCRIPT_DIR}/../env"
NETWORK_NAME="labour-bureau_internal"
POLITBURO_CONTAINER_NAME="politburo"

echo -e "${GREEN}🏛️  Starting Politburo...${NC}"

if container_exists "$POLITBURO_CONTAINER_NAME"; then
    if ! container_running "$POLITBURO_CONTAINER_NAME"; then
        echo "  Starting existing container..."
        podman start "$POLITBURO_CONTAINER_NAME"
    else
        echo "  Already running"
    fi
else
    echo "  Building and creating new container..."
    # Build image first
    podman build -t politburo:latest -f "${SCRIPT_DIR}/../../../services/politburo/Dockerfile" "${SCRIPT_DIR}/../../.."
    
    podman run -d \
        --name "$POLITBURO_CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        --network-alias "$POLITBURO_CONTAINER_NAME" \
        --env-file "${ENV_DIR}/politburo.env" \
        -p 127.0.0.1:8080:8080 \
        --restart unless-stopped \
        --health-cmd "wget --spider --quiet http://localhost:8080/healthCheck || exit 1" \
        --health-interval 30s \
        --health-timeout 5s \
        --health-retries 3 \
        politburo:latest
fi
