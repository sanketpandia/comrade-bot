#!/bin/bash
# clean-exited-for-compose.sh - Remove exited containers that match our compose service names
# so "podman compose up -d" can create/start them cleanly. Use when you see "container name already in use".

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

# Our compose container_name list (must match docker-compose.prod.yml)
SERVICES=(politburo comrade-bot db redis prometheus loki promtail grafana)
# Old "prod" project containers (legacy)
LEGACY=(prod_db_1 prod_politburo_1 prod_comrade-bot_1)

echo "Removing exited containers that may block compose..."

for name in "${SERVICES[@]}" "${LEGACY[@]}"; do
    if podman ps -a --format "{{.Names}} {{.Status}}" 2>/dev/null | grep -q "^${name} "; then
        status=$(podman inspect -f '{{.State.Status}}' "$name" 2>/dev/null || true)
        if [ "$status" = "exited" ] || [ "$status" = "dead" ]; then
            echo "  Removing exited: $name"
            podman rm -f "$name" 2>/dev/null || true
        fi
    fi
done

echo "Done. Run ./start-services.sh or podman compose up -d from prod/ to start the stack."
