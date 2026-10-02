#!/usr/bin/env bash
# Remove stale Podman/Docker dev stack containers (including fixed names from the
# old labour-bureau layout). Keeps Compose volumes unless CLEAN_VOLUMES=1.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_CMD="${CONTAINER_CLI:-docker} compose"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.dev.yml"
CLI="${CONTAINER_CLI:-docker}"

cd "$SCRIPT_DIR"

echo "Stopping Compose project (infra/dev)..."
$COMPOSE_CMD -f docker-compose.dev.yml down --remove-orphans ${CLEAN_VOLUMES:+ -v} || true

# Fixed container_name values in docker-compose.dev.yml — often left from labour-bureau.
FIXED_NAMES=(redis prometheus loki promtail grafana)
for name in "${FIXED_NAMES[@]}"; do
  if $CLI container exists "$name" 2>/dev/null; then
    echo "Removing container: $name"
    $CLI rm -f "$name" 2>/dev/null || true
  fi
done

# Legacy labour-bureau compose project (wrong bind mounts under infinite-experiment/).
for name in labour-bureau_db_1 labour-bureau_swagger-editor_1 labour-bureau_pgadmin_1; do
  if $CLI container exists "$name" 2>/dev/null; then
    echo "Removing legacy container: $name"
    $CLI rm -f "$name" 2>/dev/null || true
  fi
done

if [ "$CLI" = "podman" ]; then
  for pod in pod_dev pod_labour-bureau; do
    if podman pod exists "$pod" 2>/dev/null; then
      echo "Removing pod: $pod"
      podman pod rm -f "$pod" 2>/dev/null || true
    fi
  done
fi

echo "Done. Start fresh with:"
echo "  cd \"$SCRIPT_DIR\" && $COMPOSE_CMD -f docker-compose.dev.yml up"
