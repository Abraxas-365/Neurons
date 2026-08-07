#!/usr/bin/env bash
# Starts the Neurons API against the containers declared in docker-compose.yml.
# The Go config reads plain environment variables and there is no .env loader,
# so without these the server silently falls back to postgres/manifesto on 5432
# and appears to run while talking to the wrong database.
set -euo pipefail

export DB_HOST="${DB_HOST:-localhost}"
export DB_PORT="${DB_PORT:-5490}"
export DB_USER="${DB_USER:-neurons}"
export DB_PASSWORD="${DB_PASSWORD:-supersecret}"
export DB_NAME="${DB_NAME:-neuronsdb}"
export DB_SSL_MODE="${DB_SSL_MODE:-disable}"

export REDIS_HOST="${REDIS_HOST:-localhost}"
export REDIS_PORT="${REDIS_PORT:-6390}"

# 8080 is contested by other projects on this machine.
export SERVER_PORT="${SERVER_PORT:-8081}"

cd "$(dirname "$0")/.."
exec go run ./cmd
