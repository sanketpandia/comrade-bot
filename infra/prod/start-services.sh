#!/bin/bash
# start-services.sh - Start stack via podman compose (ad-hoc). Prefer systemd compose-stack.service on VPS.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/scripts/common.sh"

COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.prod.yml"
LOG_SHIPPER_UNIT="podman-log-shipper.service"

if [ -f "${SCRIPT_DIR}/env/monitoring.env" ]; then
    set -a
    # shellcheck source=/dev/null
    source "${SCRIPT_DIR}/env/monitoring.env"
    set +a
fi

echo -e "${GREEN}Starting production stack (podman compose)${NC}"
cd "$SCRIPT_DIR"
podman compose -f "$COMPOSE_FILE" up -d

if ! systemctl is-active --quiet "$LOG_SHIPPER_UNIT" 2>/dev/null; then
    echo "Starting ${LOG_SHIPPER_UNIT} (optional, for Promtail file shipping)..."
    sudo systemctl start "$LOG_SHIPPER_UNIT" 2>/dev/null || \
        echo "  Install: sudo cp systemd/${LOG_SHIPPER_UNIT} /etc/systemd/system/ && sudo systemctl enable --now ${LOG_SHIPPER_UNIT}"
fi

podman compose -f "$COMPOSE_FILE" ps
