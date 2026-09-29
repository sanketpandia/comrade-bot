#!/bin/bash
# start-prometheus.sh - Start Prometheus monitoring service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

NETWORK_NAME="labour-bureau_internal"
CONTAINER_NAME="prometheus"

echo -e "${GREEN}📊 Starting Prometheus...${NC}"

# Ensure network exists
if ! podman network exists "$NETWORK_NAME" 2>/dev/null; then
    echo "  Creating network: $NETWORK_NAME"
    podman network create "$NETWORK_NAME"
fi

if container_exists "$CONTAINER_NAME"; then
    # Check if container is on the correct network
    CURRENT_NETWORK=$(podman inspect "$CONTAINER_NAME" 2>/dev/null | grep -A 5 '"Networks"' | grep -o '"[^"]*":' | head -1 | tr -d '":' || echo "")
    if [ "$CURRENT_NETWORK" != "$NETWORK_NAME" ] && [ -n "$CURRENT_NETWORK" ]; then
        echo "  Container exists but on wrong network ($CURRENT_NETWORK), recreating on $NETWORK_NAME..."
        podman stop "$CONTAINER_NAME" 2>/dev/null || true
        podman rm "$CONTAINER_NAME" 2>/dev/null || true
    elif ! container_running "$CONTAINER_NAME"; then
        echo "  Starting existing container..."
        podman start "$CONTAINER_NAME"
    else
        echo "  Already running"
        # Double-check network connectivity
        if ! podman inspect "$CONTAINER_NAME" 2>/dev/null | grep -q "\"$NETWORK_NAME\""; then
            echo "  Warning: Container may not be on correct network, but continuing..."
        fi
        exit 0
    fi
fi

if ! container_exists "$CONTAINER_NAME"; then
    echo "  Creating new container..."
    podman run -d \
        --name "$CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        -p 9090:9090 \
        -v "${SCRIPT_DIR}/../prometheus.prod.yml:/etc/prometheus/prometheus.yml:ro" \
        -v labour-bureau_prometheus-storage:/prometheus \
        --restart unless-stopped \
        docker.io/prom/prometheus:latest \
        --config.file=/etc/prometheus/prometheus.yml \
        --storage.tsdb.path=/prometheus \
        --storage.tsdb.retention.time=30d
fi
