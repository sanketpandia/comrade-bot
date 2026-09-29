#!/bin/bash
# start-promtail.sh - Start Promtail log shipper service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

NETWORK_NAME="labour-bureau_internal"
CONTAINER_NAME="promtail"

echo -e "${GREEN}📤 Starting Promtail...${NC}"

if container_exists "$CONTAINER_NAME"; then
    if ! container_running "$CONTAINER_NAME"; then
        echo "  Starting existing container..."
        podman start "$CONTAINER_NAME"
    else
        echo "  Already running"
    fi
else
    echo "  Creating new container..."
    # Create log directory if it doesn't exist (needs to be writable by log shipper)
    sudo mkdir -p /var/log/containers
    sudo chmod 777 /var/log/containers 2>/dev/null || true
    
    # For Podman, we scrape container logs from files
    podman run -d \
        --name "$CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        -p 9080:9080 \
        -v "${SCRIPT_DIR}/../promtail-config.yml:/etc/promtail/config.yml:ro" \
        -v /var/log/containers:/var/log/containers:ro \
        --restart unless-stopped \
        docker.io/grafana/promtail:latest \
        -config.file=/etc/promtail/config.yml
fi
