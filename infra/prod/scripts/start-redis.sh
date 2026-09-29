#!/bin/bash
# start-redis.sh - Start Redis cache service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ENV_DIR="${SCRIPT_DIR}/../env"
NETWORK_NAME="labour-bureau_internal"
REDIS_CONTAINER_NAME="redis"

echo -e "${GREEN}🔴 Starting Redis...${NC}"

# Get Redis password from env file
if [ -f "${ENV_DIR}/cache.env" ]; then
    REDIS_PASSWORD=$(grep "^REDIS_PASSWORD=" "${ENV_DIR}/cache.env" | sed 's/^REDIS_PASSWORD=//' | tr -d '"')
else
    echo -e "  ${RED}✗ Error: cache.env not found${NC}"
    exit 1
fi

if container_exists "$REDIS_CONTAINER_NAME"; then
    if ! container_running "$REDIS_CONTAINER_NAME"; then
        echo "  Starting existing container..."
        podman start "$REDIS_CONTAINER_NAME"
    else
        echo "  Already running"
    fi
else
    echo "  Creating new container..."
    podman run -d \
        --name "$REDIS_CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        --network-alias "$REDIS_CONTAINER_NAME" \
        --env-file "${ENV_DIR}/cache.env" \
        -v labour-bureau_redis-prod:/data \
        --restart unless-stopped \
        --health-cmd "sh -c \"redis-cli --raw -a '${REDIS_PASSWORD}' incr ping\"" \
        --health-interval 10s \
        --health-timeout 3s \
        --health-retries 5 \
        docker.io/library/redis:7-alpine \
        sh -c "redis-server --appendonly yes --requirepass '${REDIS_PASSWORD}'"
fi
