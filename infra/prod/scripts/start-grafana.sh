#!/bin/bash
# start-grafana.sh - Start Grafana monitoring dashboard service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ENV_DIR="${SCRIPT_DIR}/../env"
NETWORK_NAME="labour-bureau_internal"
CONTAINER_NAME="grafana"

echo -e "${GREEN}📈 Starting Grafana...${NC}"

# Get Grafana password from env file
if [ -f "${ENV_DIR}/monitoring.env" ]; then
    GRAFANA_PASSWORD=$(grep "^GRAFANA_ADMIN_PASSWORD=" "${ENV_DIR}/monitoring.env" | sed 's/^GRAFANA_ADMIN_PASSWORD=//' | tr -d '"')
else
    GRAFANA_PASSWORD="admin"
fi

if container_exists "$CONTAINER_NAME"; then
    if ! container_running "$CONTAINER_NAME"; then
        echo "  Starting existing container..."
        podman start "$CONTAINER_NAME"
    else
        echo "  Already running"
    fi
else
    echo "  Building custom Grafana image..."
    GRAFANA_DIR="${SCRIPT_DIR}/../grafana"
    podman build -t grafana-custom:latest -f "${GRAFANA_DIR}/Dockerfile" "${GRAFANA_DIR}"
    
    echo "  Creating new container..."
    podman run -d \
        --name "$CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        -p 127.0.0.1:3000:3000 \
        --env-file "${ENV_DIR}/monitoring.env" \
        -e GF_SECURITY_ADMIN_PASSWORD="${GRAFANA_PASSWORD}" \
        -e GF_SECURITY_ADMIN_USER=admin \
        -v labour-bureau_grafana-storage:/var/lib/grafana \
        --restart unless-stopped \
        grafana-custom:latest
fi