#!/bin/bash
# podman-logs-to-files.sh - Tail Podman container logs and write to files for Promtail

LOG_DIR="/var/log/containers"
mkdir -p "$LOG_DIR"

# Function to tail a container's logs
tail_container_logs() {
    local container_name=$1
    local log_file="${LOG_DIR}/${container_name}.log"
    
    echo "Starting log tail for container: $container_name -> $log_file"
    
    while true; do
        if podman ps --format "{{.Names}}" | grep -q "^${container_name}$"; then
            # Container is running, tail its logs
            podman logs -f "$container_name" 2>&1 | while IFS= read -r line; do
                echo "[$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")] $line" >> "$log_file"
            done
        else
            # Container is not running, wait and check again
            sleep 5
        fi
    done
}

# Get list of containers to monitor
CONTAINERS=("politburo" "comrade-bot" "db" "redis" "loki" "promtail" "grafana" "prometheus")

# Start tailing logs for each container in background
for container in "${CONTAINERS[@]}"; do
    tail_container_logs "$container" &
done

# Wait for all background processes
wait
