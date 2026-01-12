#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "Starting PostgreSQL container..."
docker compose up -d postgres

echo "Waiting for PostgreSQL to be ready..."
timeout 30s bash -c '
    while ! docker inspect gofast-postgres --format="{{.State.Health.Status}}" 2>/dev/null | grep -q "healthy"; do
        sleep 1
    done
'

echo "Updating user permissions to admin..."
docker exec gofast-postgres psql -U postgres -d postgres -c "UPDATE users SET access = -1;"

echo "Done."
