#!/usr/bin/env bash
# Pull-deploy script for the BACKEND, run on the VPS (either manually or
# via a CI workflow over SSH). Expects to live at /opt/neurons, alongside a
# gitignored .env holding runtime secrets (DB_PASSWORD, JWT_SECRET_KEY,
# OAUTH_GOOGLE_*, etc.)
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

# Pull first, then re-exec the freshly pulled copy of this script. Without
# the re-exec, bash keeps reading this file at stale byte offsets after
# `git reset --hard` rewrites it mid-run (bash reads scripts incrementally),
# executing a hybrid of old and new versions whenever deploy.sh changes.
if [ "${1:-}" != "--updated" ]; then
    echo "==> Pulling latest main"
    git fetch origin main
    git reset --hard origin/main
    exec bash "${BASH_SOURCE[0]}" --updated
fi

if [ ! -f .env ]; then
    echo "ERROR: .env not found in $(pwd). Copy it from a backup or set it up before deploying." >&2
    exit 1
fi
set -a
source .env
set +a

echo "==> Building backend image"
docker compose -f docker-compose.prod.yml build api

echo "==> Starting postgres/redis (if not already up)"
docker compose -f docker-compose.prod.yml up -d postgres redis

echo "==> Waiting for postgres to be healthy"
for i in $(seq 1 30); do
    status=$(docker inspect -f '{{.State.Health.Status}}' neurons-postgres 2>/dev/null || echo "starting")
    [ "$status" = "healthy" ] && break
    sleep 2
done

echo "==> Running database migrations"
DB_HOST=localhost ./scripts/migrate.sh

echo "==> Building frontend"
rm -rf /tmp/neurons-frontend-dist
docker build \
    --build-arg VITE_API_URL="${PUBLIC_API_URL:-}" \
    -t neurons-frontend:latest \
    -f frontend/Dockerfile \
    --output type=local,dest=/tmp/neurons-frontend-dist \
    frontend

rm -rf frontend/dist
mv /tmp/neurons-frontend-dist/dist frontend/dist
rmdir /tmp/neurons-frontend-dist 2>/dev/null || true
chmod -R a+rX frontend/dist

echo "==> Restarting backend"
docker compose -f docker-compose.prod.yml up -d api

echo "==> Reloading Caddy"
sudo systemctl reload caddy

echo "==> Cleaning up unused images"
docker image prune -f >/dev/null

echo "==> Deploy complete: $(git rev-parse --short HEAD)"
