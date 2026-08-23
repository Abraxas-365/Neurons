#!/usr/bin/env bash
# Starts the built Neurons API (./bin/server) with environment from ../.env.
# The Go config reads plain env vars only — there is no .env loader in the code.
set -euo pipefail
cd "$(dirname "$0")/.."
set -a
. ./.env
set +a
exec ./bin/server
