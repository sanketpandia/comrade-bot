#!/bin/bash
# watchdog.sh - Check container health and restart dead containers
# This script is called periodically by the systemd timer to ensure
# all containers are running. It fixes the gap where rootless podman's
# --restart policy may not work reliably.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/scripts/common.sh"

CONTAINERS=("db" "redis" "politburo" "comrade-bot" "node-exporter" "prometheus" "loki" "promtail" "grafana")
RESTART_NEEDED=false

for container in "${CONTAINERS[@]}"; do
    if container_exists "$container"; then
        if ! container_running "$container"; then
            echo "$(date '+%Y-%m-%d %H:%M:%S') ⚠️  Container '$container' exists but is not running. Restarting..."
            podman start "$container"
            if [ $? -eq 0 ]; then
                echo "$(date '+%Y-%m-%d %H:%M:%S') ✅ Container '$container' restarted successfully"
            else
                echo "$(date '+%Y-%m-%d %H:%M:%S') ❌ Failed to restart container '$container'"
            fi
            RESTART_NEEDED=true
        fi
    else
        echo "$(date '+%Y-%m-%d %H:%M:%S') ❌ Container '$container' does not exist! Running full start..."
        "${SCRIPT_DIR}/start-services.sh"
        exit $?
    fi
done

if [ "$RESTART_NEEDED" = false ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') ✅ All containers healthy"
fi
