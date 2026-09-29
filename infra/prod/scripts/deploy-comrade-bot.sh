#!/bin/bash
# deploy-comrade-bot.sh - Deploy Comrade Bot via podman compose (rebuild and recreate)
# Usage: ./deploy-comrade-bot.sh [local|global]
#   local  - Deploy commands to guild (requires GUILD_ID in env)
#   global - Deploy commands globally (default if no argument)
# Set ALLOW_COMMAND_DEPLOY_FAILURE=true to keep the bot restart non-blocking if
# Discord command deployment fails and commands will be deployed manually.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROD_DIR="${SCRIPT_DIR}/.."
source "${SCRIPT_DIR}/common.sh"

COMPOSE_FILE="${PROD_DIR}/docker-compose.prod.yml"
COMRADE_BOT_CONTAINER_NAME="comrade-bot"

DEPLOY_MODE="${1:-global}"

echo -e "${GREEN}🚀 Deploying Comrade Bot...${NC}"

cd "$PROD_DIR"
podman compose -f "$COMPOSE_FILE" build comrade-bot
podman compose -f "$COMPOSE_FILE" up -d comrade-bot

echo "  Waiting for container to be ready..."
sleep 3

# Verify command build and deploy Discord commands
if ! podman exec "$COMRADE_BOT_CONTAINER_NAME" test -f dist/deploy-commands.js 2>/dev/null; then
    echo -e "${YELLOW}⚠ Warning: dist/deploy-commands.js not found. Building commands...${NC}"
    podman exec "$COMRADE_BOT_CONTAINER_NAME" npm run build || {
        echo -e "${RED}✗ Error: Failed to build commands${NC}"
        exit 1
    }
fi

echo "  Deploying Discord commands (mode: $DEPLOY_MODE)..."
if [ "$DEPLOY_MODE" = "local" ]; then
    COMMAND_DEPLOY_SCRIPT="commands:deploy:local"
else
    COMMAND_DEPLOY_SCRIPT="commands:deploy:global"
fi

if ! podman exec "$COMRADE_BOT_CONTAINER_NAME" npm run "$COMMAND_DEPLOY_SCRIPT"; then
    echo -e "${RED}✗ Error: Command deployment failed.${NC}"
    echo -e "${YELLOW}Remediation: podman exec $COMRADE_BOT_CONTAINER_NAME npm run $COMMAND_DEPLOY_SCRIPT${NC}"

    if [ "${ALLOW_COMMAND_DEPLOY_FAILURE:-false}" = "true" ]; then
        echo -e "${YELLOW}⚠ ALLOW_COMMAND_DEPLOY_FAILURE=true; continuing despite command deployment failure.${NC}"
    else
        exit 1
    fi
fi

echo -e "${GREEN}✅ Comrade Bot deployed successfully!${NC}"
