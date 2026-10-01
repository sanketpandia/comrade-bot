#!/bin/bash
# deploy-caddy.sh - Validate edge/Caddyfile and reload caddy-rootful.service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CADDYFILE="${SCRIPT_DIR}/Caddyfile"
SERVICE_NAME="caddy-rootful.service"

echo "Validating ${CADDYFILE}..."
if ! caddy validate --config "$CADDYFILE" --adapter caddyfile; then
    exit 1
fi

if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "Reloading ${SERVICE_NAME}..."
    sudo systemctl reload "$SERVICE_NAME" || sudo systemctl restart "$SERVICE_NAME"
else
    echo "Starting ${SERVICE_NAME}..."
    sudo systemctl start "$SERVICE_NAME"
fi

sudo systemctl status "$SERVICE_NAME" --no-pager -l
