#!/bin/bash
# start-services.sh - Start the Labour Bureau stack via podman compose (ad-hoc/manual)
# For production, use the systemd unit labour-bureau-compose.service instead, which
# runs "podman compose up" in the foreground so the stack stays up reliably.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/scripts/common.sh"

COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.prod.yml"

echo -e "${GREEN}🚀 Starting Labour Bureau stack (podman compose)${NC}"

# Load GRAFANA_ADMIN_PASSWORD for compose substitution if present
if [ -f "${SCRIPT_DIR}/env/monitoring.env" ]; then
    set -a
    source "${SCRIPT_DIR}/env/monitoring.env"
    set +a
fi

cd "$SCRIPT_DIR"
podman compose -f "$COMPOSE_FILE" up -d

# Start log shipper systemd service if not running
if ! systemctl is-active --quiet podman-log-shipper.service 2>/dev/null; then
    echo "📝 Starting Podman Log Shipper service..."
    sudo systemctl start podman-log-shipper.service 2>/dev/null || echo "  Note: Install with: sudo systemctl enable --now $(realpath "${SCRIPT_DIR}/podman-log-shipper.service")"
fi

echo -e "\n${GREEN}✅ Stack started!${NC}"
echo ""
echo "📊 Service Status:"
podman compose -f "$COMPOSE_FILE" ps
