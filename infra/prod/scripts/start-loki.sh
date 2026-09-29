#!/bin/bash
# start-loki.sh - Start Loki logging service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

NETWORK_NAME="labour-bureau_internal"
CONTAINER_NAME="loki"

echo -e "${GREEN}📝 Starting Loki...${NC}"

if container_exists "$CONTAINER_NAME"; then
    if ! container_running "$CONTAINER_NAME"; then
        echo "  Starting existing container..."
        podman start "$CONTAINER_NAME"
    else
        echo "  Already running"
    fi
else
    echo "  Creating new container..."
    podman run -d \
        --name "$CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        -p 3100:3100 \
        -v "${SCRIPT_DIR}/../loki.prod.yml:/etc/loki/local-config.yml:ro" \
        -v labour-bureau_loki-storage:/loki \
        --restart unless-stopped \
        docker.io/grafana/loki:latest \
        -config.file=/etc/loki/local-config.yml
fi
