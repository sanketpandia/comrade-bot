#!/bin/bash
# common.sh - Common functions and variables for service scripts

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check if container exists
container_exists() {
    podman ps -a --format "{{.Names}}" | grep -q "^${1}$"
}

# Function to check if container is running
container_running() {
    podman ps --format "{{.Names}}" | grep -q "^${1}$"
}
