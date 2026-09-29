#!/bin/bash
# stop-services.sh - Stop the Labour Bureau stack via podman compose

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.prod.yml"

echo "🛑 Stopping Labour Bureau stack..."

cd "$SCRIPT_DIR"
podman compose -f "$COMPOSE_FILE" down

echo "✅ Stack stopped"
