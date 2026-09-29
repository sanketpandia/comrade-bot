#!/bin/bash
# deploy-caddy.sh - Deploy Caddy configuration and restart service

set -e

# Ensure we're running with bash
if [ -z "$BASH_VERSION" ]; then
    echo "❌ Error: This script requires bash. Run with: bash $0" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CADDYFILE_SOURCE="${SCRIPT_DIR}/Caddyfile"
CADDYFILE_DEST="/etc/caddy/Caddyfile"
SERVICE_NAME="caddy.service"

echo "🚀 Deploying Caddy configuration..."

# Check if Caddyfile exists
if [ ! -f "$CADDYFILE_SOURCE" ]; then
    echo "❌ Error: Caddyfile not found at $CADDYFILE_SOURCE"
    exit 1
fi

# Validate Caddyfile before deploying
echo "📋 Validating Caddyfile..."
if ! caddy validate --config "$CADDYFILE_SOURCE" --adapter caddyfile > /dev/null 2>&1; then
    echo "❌ Error: Caddyfile validation failed!"
    echo "Run: caddy validate --config $CADDYFILE_SOURCE --adapter caddyfile"
    exit 1
fi
echo "✅ Caddyfile is valid"

# Create /etc/caddy directory if it doesn't exist
echo "📁 Creating /etc/caddy directory..."
sudo mkdir -p /etc/caddy

# Backup existing Caddyfile if it exists
if [ -f "$CADDYFILE_DEST" ]; then
    echo "💾 Backing up existing Caddyfile..."
    sudo cp "$CADDYFILE_DEST" "${CADDYFILE_DEST}.backup.$(date +%Y%m%d_%H%M%S)"
fi

# Copy Caddyfile to central location
echo "📝 Copying Caddyfile to $CADDYFILE_DEST..."
sudo cp "$CADDYFILE_SOURCE" "$CADDYFILE_DEST"
sudo chown root:root "$CADDYFILE_DEST"
sudo chmod 644 "$CADDYFILE_DEST"

# Reload Caddy service (graceful reload if running, start if not)
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "🔄 Reloading Caddy service (graceful reload)..."
    sudo systemctl reload "$SERVICE_NAME" || {
        echo "⚠️  Reload failed, restarting service..."
        sudo systemctl restart "$SERVICE_NAME"
    }
else
    echo "▶️  Starting Caddy service..."
    sudo systemctl start "$SERVICE_NAME"
fi

# Wait a moment and check status
sleep 2
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "✅ Caddy service is running"
    echo ""
    echo "📊 Service status:"
    sudo systemctl status "$SERVICE_NAME" --no-pager -l
else
    echo "❌ Error: Caddy service failed to start!"
    echo ""
    echo "📋 Recent logs:"
    sudo journalctl -u "$SERVICE_NAME" -n 20 --no-pager
    exit 1
fi
