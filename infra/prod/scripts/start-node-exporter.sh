#!/bin/bash
# start-node-exporter.sh - Start Node Exporter for system metrics

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

NETWORK_NAME="labour-bureau_internal"
CONTAINER_NAME="node-exporter"

echo -e "${GREEN}🖥️  Starting Node Exporter...${NC}"

# Ensure network exists
if ! podman network exists "$NETWORK_NAME" 2>/dev/null; then
    echo "  Creating network: $NETWORK_NAME"
    podman network create "$NETWORK_NAME"
fi

if container_exists "$CONTAINER_NAME"; then
    if ! container_running "$CONTAINER_NAME"; then
        echo "  Starting existing container..."
        podman start "$CONTAINER_NAME"
    else
        echo "  Already running"
        exit 0
    fi
else
    echo "  Creating new container..."
    # Node Exporter needs access to host system files for metrics
    # Using --pid=host and mounting /proc, /sys to expose system metrics
    podman run -d \
        --name "$CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        --pid host \
        -p 9100:9100 \
        -v /proc:/host/proc:ro \
        -v /sys:/host/sys:ro \
        -v /:/rootfs:ro \
        --restart unless-stopped \
        docker.io/prom/node-exporter:latest \
        --path.procfs=/host/proc \
        --path.sysfs=/host/sys \
        --collector.filesystem.mount-points-exclude="^/(sys|proc|dev|host|etc)($$|/)"
fi
