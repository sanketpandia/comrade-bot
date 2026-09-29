#!/bin/bash
# restart-services.sh - Restart all services

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🔄 Restarting Labour Bureau Services..."

# Stop services
"${SCRIPT_DIR}/stop-services.sh"

# Wait a moment
sleep 2

# Start services
"${SCRIPT_DIR}/start-services.sh"
