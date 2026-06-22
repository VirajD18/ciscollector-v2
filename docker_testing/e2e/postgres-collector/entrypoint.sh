#!/bin/bash
set -euo pipefail

export MAIN_SERVER_URL="${MAIN_SERVER_URL:-http://main-server:8081}"
export COLLECTOR_TOKEN="${COLLECTOR_TOKEN:-e2e-test-collector-token-do-not-use-in-prod}"
export COLLECTOR_SCHEDULE="${COLLECTOR_SCHEDULE:-*/5 * * * *}"
export SCAN_COMMANDS="${SCAN_COMMANDS:-postgres_cis,hba_scanner,pii_scanner,ssl_audit,inactive_users,unique_ip,unused_lines,password_leak_scanner}"
export PII_RUN_OPTION="${PII_RUN_OPTION:-datascan}"
export PII_SCHEDULE="${PII_SCHEDULE:-}"
export LOG_PARSER_SCHEDULE="${LOG_PARSER_SCHEDULE:-}"
export LOG_PARSER_PREFIX="${LOG_PARSER_PREFIX:-%t %u %d %h }"
export LOG_PARSER_LOGFILE="${LOG_PARSER_LOGFILE:-/var/lib/postgresql/data/log/*.log}"
export LOG_PARSER_HBAFILE="${LOG_PARSER_HBAFILE:-/var/lib/postgresql/data/pg_hba.conf}"
# Unique per scaled replica (docker compose names containers e2e-collector-1, e2e-collector-2, …).
export APP_HOSTNAME="${APP_HOSTNAME:-$(hostname -f 2>/dev/null || hostname)}"
export POSTGRES_HOST="${POSTGRES_HOST:-${APP_HOSTNAME}}"

# Map unique host label to local Postgres (target_id uses POSTGRES_HOST; DB listens on 127.0.0.1).
sed -i "/[[:space:]]${POSTGRES_HOST}$/d" /etc/hosts 2>/dev/null || true
echo "127.0.0.1 ${POSTGRES_HOST}" >> /etc/hosts
export PGPASSWORD="${POSTGRES_PASSWORD}"

mkdir -p /etc/klouddbshield
envsubst < /app/kshieldconfig.toml.template > /etc/klouddbshield/kshieldconfig.toml

echo "Rendered kshieldconfig.toml for host ${APP_HOSTNAME}"

wait_for_postgres() {
  until psql -h "${POSTGRES_HOST}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -c "SELECT 1" >/dev/null 2>&1; do
    echo "Waiting for PostgreSQL at ${POSTGRES_HOST} (post-init)..."
    sleep 1
  done
}

wait_for_main_server() {
  local url="${MAIN_SERVER_URL%/}/api/overview"
  until curl -sf "${url}" >/dev/null 2>&1; do
    echo "Waiting for main-server at ${url}..."
    sleep 2
  done
}

# Start PostgreSQL using the official entrypoint (runs initdb scripts on first boot).
/usr/local/bin/docker-entrypoint.sh postgres &
PG_PID=$!

wait_for_postgres
wait_for_main_server

echo "Running initial scans (CIS, HBA, PII, SSL, log parser all) and pushing to main-server..."
# --json enables main-server push; --output-type json avoids text-format nil panics in Docker.
/app/ciscollector --config /etc/klouddbshield --json --output-type json \
  --run-postgres --hba-scanner --piiscanner --ssl-check \
  --logparser all \
  --prefix "${LOG_PARSER_PREFIX}" \
  --file-path "${LOG_PARSER_LOGFILE}" \
  --hba-file "${LOG_PARSER_HBAFILE}" || {
  echo "Initial scan failed; continuing with cron mode"
}

echo "Starting ciscollector cron (--setup-cron, same as systemd ciscollector.service)..."
trap 'kill -TERM "${PG_PID}" 2>/dev/null || true' EXIT INT TERM

exec /app/ciscollector --setup-cron --config /etc/klouddbshield --json
