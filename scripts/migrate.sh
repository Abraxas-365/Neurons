#!/usr/bin/env bash
# Applies any migrations/*.up.sql files not yet recorded in schema_migrations,
# in filename order. Safe to run on every deploy - already-applied files are
# skipped. Expects standard PG* / DB_* env vars (or PGHOST/PGPORT/PGUSER/
# PGPASSWORD/PGDATABASE) to already be set in the environment.
set -euo pipefail

MIGRATIONS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/migrations"

PGHOST="${PGHOST:-${DB_HOST:-localhost}}"
PGPORT="${PGPORT:-${DB_PORT:-5432}}"
PGUSER="${PGUSER:-${DB_USER:-neurons}}"
PGPASSWORD="${PGPASSWORD:-${DB_PASSWORD:-}}"
PGDATABASE="${PGDATABASE:-${DB_NAME:-neuronsdb}}"
export PGHOST PGPORT PGUSER PGPASSWORD PGDATABASE

psql -v ON_ERROR_STOP=1 -q -c "
CREATE TABLE IF NOT EXISTS schema_migrations (
    filename    TEXT PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
"

applied=0
for file in $(ls "$MIGRATIONS_DIR"/*.up.sql | sort); do
    name="$(basename "$file")"
    already=$(psql -tA -c "SELECT 1 FROM schema_migrations WHERE filename = '$name'")
    if [ "$already" = "1" ]; then
        continue
    fi
    echo "Applying migration: $name"
    psql -v ON_ERROR_STOP=1 -q -1 -f "$file"
    psql -v ON_ERROR_STOP=1 -q -c "INSERT INTO schema_migrations (filename) VALUES ('$name')"
    applied=$((applied + 1))
done

echo "Migrations complete: $applied applied"
