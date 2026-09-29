#!/bin/bash
# start-comrade-bot.sh - Start Comrade Bot service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ENV_DIR="${SCRIPT_DIR}/../env"
NETWORK_NAME="labour-bureau_internal"
COMRADE_BOT_CONTAINER_NAME="comrade-bot"

echo -e "${GREEN}🤖 Starting Comrade Bot...${NC}"

if container_exists "$COMRADE_BOT_CONTAINER_NAME"; then
    if ! container_running "$COMRADE_BOT_CONTAINER_NAME"; then
        echo "  Starting existing container..."
        podman start "$COMRADE_BOT_CONTAINER_NAME"
    else
        echo "  Already running"
    fi
else
    echo "  Building and creating new container..."
    # Build image first
    podman build -t comrade-bot:latest -f "${SCRIPT_DIR}/../../../services/comrade-bot-discord/Dockerfile" "${SCRIPT_DIR}/../../.."
    
    # Get DISCORD_BOT_TOKEN from env file (or BOT_TOKEN as fallback)
    if [ -f "${ENV_DIR}/comrade-bot.env" ]; then
        DISCORD_BOT_TOKEN=$(grep "^DISCORD_BOT_TOKEN=" "${ENV_DIR}/comrade-bot.env" | sed 's/^DISCORD_BOT_TOKEN=//' | tr -d '"')
        # Fallback to BOT_TOKEN if DISCORD_BOT_TOKEN not found
        if [ -z "$DISCORD_BOT_TOKEN" ]; then
            DISCORD_BOT_TOKEN=$(grep "^BOT_TOKEN=" "${ENV_DIR}/comrade-bot.env" | sed 's/^BOT_TOKEN=//' | tr -d '"')
        fi
        if [ -z "$DISCORD_BOT_TOKEN" ]; then
            echo -e "  ${RED}✗ Error: DISCORD_BOT_TOKEN or BOT_TOKEN not found in comrade-bot.env${NC}"
            exit 1
        fi
    else
        echo -e "  ${RED}✗ Error: comrade-bot.env not found${NC}"
        exit 1
    fi
    
    podman run -d \
        --name "$COMRADE_BOT_CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        --env-file "${ENV_DIR}/comrade-bot.env" \
        -e DISCORD_BOT_TOKEN="${DISCORD_BOT_TOKEN}" \
        --restart unless-stopped \
        comrade-bot:latest
fi
