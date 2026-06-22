#!/bin/bash
# Start e2e stack with PostgreSQL as main-server storage (not SQLite).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"

export KSHIELD_DB_DRIVER="${KSHIELD_DB_DRIVER:-postgres}"
export MAIN_SERVER_DATABASE_URL="${MAIN_SERVER_DATABASE_URL:-postgres://kshield:kshield@main-server-db:5432/kshield?sslmode=disable}"

exec "${ROOT}/run.sh" "$@"
