#!/bin/bash
# start-db.sh - Start PostgreSQL database service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ENV_DIR="${SCRIPT_DIR}/../env"
NETWORK_NAME="labour-bureau_internal"
DB_CONTAINER_NAME="db"

echo -e "${GREEN}🗄️  Starting PostgreSQL...${NC}"

if container_exists "$DB_CONTAINER_NAME"; then
    if ! container_running "$DB_CONTAINER_NAME"; then
        echo "  Starting existing container..."
        podman start "$DB_CONTAINER_NAME"
    else
        echo "  Already running"
    fi
else
    echo "  Creating new container..."
    podman run -d \
        --name "$DB_CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        --network-alias "$DB_CONTAINER_NAME" \
        --env-file "${ENV_DIR}/database.env" \
        -v labour-bureau_pgdata-prod:/var/lib/postgresql/data \
        -v "${SCRIPT_DIR}/../../../services/politburo/migrations:/migrations" \
        -p 127.0.0.1:5432:5432 \
        --restart unless-stopped \
        --health-cmd "pg_isready -U ieuser -d infinite" \
        --health-interval 30s \
        --health-timeout 5s \
        --health-retries 5 \
        docker.io/library/postgres:15
fi

# Wait for database to be ready
echo "  Waiting for database to be ready..."
sleep 5
for i in {1..30}; do
    if podman exec "$DB_CONTAINER_NAME" pg_isready -U ieuser -d infinite >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓ Database is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "  ${RED}✗ Database failed to start${NC}"
        exit 1
    fi
    sleep 1
done

# Run migrations if database is empty
echo "  Checking if migrations need to be run..."
MIGRATION_FILE="${SCRIPT_DIR}/../../../services/politburo/migrations/000_infinite_schema.sql"
if [ -f "$MIGRATION_FILE" ]; then
    # Check if tables exist (quick check for any table)
    TABLE_COUNT=$(podman exec "$DB_CONTAINER_NAME" psql -U ieuser -d infinite -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null || echo "0")
    
    if [ "$TABLE_COUNT" = "0" ] || [ -z "$TABLE_COUNT" ]; then
        echo "  Running database migrations..."
        podman exec -i "$DB_CONTAINER_NAME" psql -U ieuser -d infinite < "$MIGRATION_FILE"
        if [ $? -eq 0 ]; then
            echo -e "  ${GREEN}✓ Migrations completed${NC}"
        else
            echo -e "  ${YELLOW}⚠ Migration had errors (database may already be initialized)${NC}"
        fi
    else
        echo -e "  ${GREEN}✓ Database already has tables (${TABLE_COUNT} tables found), skipping migrations${NC}"
    fi
else
    echo -e "  ${YELLOW}⚠ Migration file not found: $MIGRATION_FILE${NC}"
fi
