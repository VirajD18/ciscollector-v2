#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "${ROOT}"

read_env_var() {
  local key="$1"
  [ -f .env ] || return 0
  local line
  line="$(grep -E "^${key}=" .env | tail -1 || true)"
  [ -n "${line}" ] || return 0
  echo "${line#*=}" | tr -d '"'
}

COLLECTOR_COUNT="${COLLECTOR_COUNT:-$(read_env_var COLLECTOR_COUNT)}"
COLLECTOR_COUNT="${COLLECTOR_COUNT:-2}"
KSHIELD_DB_DRIVER="${KSHIELD_DB_DRIVER:-$(read_env_var KSHIELD_DB_DRIVER)}"
KSHIELD_DB_DRIVER="${KSHIELD_DB_DRIVER:-sqlite}"

COMPOSE_PROFILES=()
if [ "${KSHIELD_DB_DRIVER}" = "postgres" ] || [ "${KSHIELD_DB_DRIVER}" = "postgresql" ] || [ "${KSHIELD_DB_DRIVER}" = "pg" ]; then
  COMPOSE_PROFILES=(--profile postgres-storage)
fi
if ! [[ "${COLLECTOR_COUNT}" =~ ^[0-9]+$ ]] || [ "${COLLECTOR_COUNT}" -lt 1 ]; then
  echo "COLLECTOR_COUNT must be a positive integer (got: ${COLLECTOR_COUNT})" >&2
  exit 1
fi
if [ "${COLLECTOR_COUNT}" -gt 100 ]; then
  echo "COLLECTOR_COUNT capped at 100 for local testing (got: ${COLLECTOR_COUNT})" >&2
  exit 1
fi

if [ "$#" -eq 0 ]; then
  set -- up --build -d --scale "collector=${COLLECTOR_COUNT}"
elif [ "${1}" = "up" ] && [[ " $* " != *" --scale "* ]]; then
  set -- "$@" --scale "collector=${COLLECTOR_COUNT}"
fi

if [ ${#COMPOSE_PROFILES[@]} -gt 0 ]; then
  docker compose --env-file .env "${COMPOSE_PROFILES[@]}" "$@"
else
  docker compose --env-file .env "$@"
fi

if [ "${1}" = "up" ]; then
  echo ""
  echo "Dashboard:  http://localhost:8081/"
  echo "API:        http://localhost:8081/api/overview"
  echo "Collectors: ${COLLECTOR_COUNT} postgres container(s) (service: collector, scaled)"
  echo "Storage:    main-server ${KSHIELD_DB_DRIVER} (collector DBs are separate Postgres instances)"
  if [ "${KSHIELD_DB_DRIVER}" != "sqlite" ]; then
    echo "            postgres URL from MAIN_SERVER_DATABASE_URL / DATABASE_URL"
  fi
  echo ""
  echo "Logs:  docker compose logs -f collector"
  echo "       docker compose logs -f main-server"
  echo "Scale: set COLLECTOR_COUNT in .env, then ./run.sh down && ./run.sh"
fi
