#!/bin/bash
# deploy-services.sh - Deploy one or more services (rebuild and restart)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/scripts/common.sh"

# Parse arguments
SERVICES=()

if [ $# -eq 0 ]; then
    # No arguments - show usage
    echo "Usage: $0 [politburo|comrade-bot|jobhunt|all]"
    echo ""
    echo "Examples:"
    echo "  $0 politburo          # Deploy only Politburo"
    echo "  $0 comrade-bot        # Deploy only Comrade Bot"
    echo "  $0 jobhunt            # Deploy only Jobhunt"
    echo "  $0 all                # Deploy all services"
    exit 1
fi

# Parse service arguments
for arg in "$@"; do
    case "$arg" in
        politburo|comrade-bot|jobhunt|all)
            SERVICES+=("$arg")
            ;;
        *)
            echo -e "${RED}✗ Unknown service: $arg${NC}"
            echo "Valid services: politburo, comrade-bot, jobhunt, all"
            exit 1
            ;;
    esac
done

# Expand "all" to both services
if [[ " ${SERVICES[@]} " =~ " all " ]]; then
    SERVICES=("politburo" "comrade-bot" "jobhunt")
fi

echo -e "${GREEN}🚀 Deploying services: ${SERVICES[*]}${NC}"
echo ""

# Deploy each service
for service in "${SERVICES[@]}"; do
    case "$service" in
        politburo)
            "${SCRIPT_DIR}/scripts/deploy-politburo.sh"
            echo ""
            ;;
        comrade-bot)
            "${SCRIPT_DIR}/scripts/deploy-comrade-bot.sh"
            echo ""
            ;;
        jobhunt)
            "${SCRIPT_DIR}/scripts/deploy-jobhunt.sh"
            echo ""
            ;;
    esac
done

echo -e "${GREEN}✅ Deployment complete!${NC}"
echo ""
echo "📊 Service Status:"
cd "$SCRIPT_DIR"
podman compose -f docker-compose.prod.yml ps 2>/dev/null || podman ps --filter "name=politburo\|comrade-bot" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
