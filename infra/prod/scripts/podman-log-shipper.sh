#!/usr/bin/env bash
# podman-log-shipper.sh - Tail Podman container logs and write to files for Promtail

# Find podman in PATH or use common locations
PODMAN=$(which podman 2>/dev/null || echo "/usr/bin/podman")

LOG_DIR="/var/log/containers"
mkdir -p "$LOG_DIR" || {
    echo "ERROR: Cannot create log directory $LOG_DIR" >&2
    exit 1
}

# Ensure log directory is writable by runner user
# Try to set permissions, but don't fail if we can't (might need sudo)
chmod 777 "$LOG_DIR" 2>/dev/null || true

# Function to tail a container's logs
tail_container_logs() {
    local container_name=$1
    local log_file="${LOG_DIR}/${container_name}.log"
    local initial_scrape=true
    
    echo "[$(date +"%Y-%m-%d %H:%M:%S")] Starting log tail for container: $container_name -> $log_file" >&2
    
    while true; do
        if $PODMAN ps --format "{{.Names}}" 2>/dev/null | grep -q "^${container_name}$"; then
            # Container is running
            if [ "$initial_scrape" = true ] && [ ! -f "$log_file" ]; then
                # Initial scrape: get all logs from the start
                echo "[$(date +"%Y-%m-%d %H:%M:%S")] Initial scrape: fetching all logs for $container_name" >&2
                $PODMAN logs "$container_name" 2>&1 | while IFS= read -r line || [ -n "$line" ]; do
                    if [ -n "$line" ]; then
                        echo "$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ") $line" >> "$log_file"
                    fi
                done
                initial_scrape=false
                echo "[$(date +"%Y-%m-%d %H:%M:%S")] Initial scrape complete for $container_name, switching to incremental polling" >&2
            else
                # Incremental scrape: get new logs since last check (2 seconds)
                $PODMAN logs --since 2s "$container_name" 2>&1 | while IFS= read -r line || [ -n "$line" ]; do
                    if [ -n "$line" ]; then
                        echo "$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ") $line" >> "$log_file"
                    fi
                done
            fi
            sleep 1
        else
            # Container is not running, wait and check again
            # Reset initial_scrape flag in case container restarts
            if [ ! -f "$log_file" ]; then
                initial_scrape=true
            fi
            sleep 5
        fi
    done
}

# Get list of containers to monitor (you can customize this list)
CONTAINERS=("politburo" "comrade-bot" "db" "redis" "loki" "promtail" "grafana" "prometheus")

echo "[$(date +"%Y-%m-%d %H:%M:%S")] Podman Log Shipper starting..." >&2
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Monitoring containers: ${CONTAINERS[*]}" >&2
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Log directory: $LOG_DIR" >&2

# Start tailing logs for each container in background
for container in "${CONTAINERS[@]}"; do
    tail_container_logs "$container" &
    sleep 1
done

# Wait for all background processes
wait
