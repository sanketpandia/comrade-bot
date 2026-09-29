#!/bin/bash
# Deploy slash commands to Discord
# Usage:
#   ./deploy.sh              - Auto-detect mode (local if GUILD_ID, else global)
#   ./deploy.sh local        - Deploy to single guild (instant)
#   ./deploy.sh global       - Deploy to all servers (~1 hour)

set -e

MODE=$1

echo "🚀 Discord Command Deployment Script"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check if running in production (compiled) or development
if [ -f "dist/deploy-commands.js" ]; then
    echo "📦 Mode: Production (using compiled JavaScript)"
    DEPLOY_CMD="node dist/deploy-commands.js"
else
    echo "🔧 Mode: Development (using TypeScript)"
    DEPLOY_CMD="npx ts-node src/deploy-commands.ts"
fi

# Run deployment with mode argument
if [ -z "$MODE" ]; then
    echo "🎯 Auto-detecting deployment scope..."
    $DEPLOY_CMD
elif [ "$MODE" = "local" ]; then
    echo "🏠 Local deployment (guild-specific)"
    $DEPLOY_CMD local
elif [ "$MODE" = "global" ]; then
    echo "🌍 Global deployment (all servers)"
    $DEPLOY_CMD global
else
    echo "❌ Invalid mode: $MODE"
    echo "   Valid modes: local, global"
    exit 1
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Deployment complete!"
