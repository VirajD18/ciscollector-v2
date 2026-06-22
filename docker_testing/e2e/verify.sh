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

BASE_URL="${BASE_URL:-http://localhost:8081}"
EXPECTED_COLLECTORS="${EXPECTED_COLLECTORS:-${COLLECTOR_COUNT:-$(read_env_var COLLECTOR_COUNT)}}"
EXPECTED_COLLECTORS="${EXPECTED_COLLECTORS:-2}"
TIMEOUT_SEC="${TIMEOUT_SEC:-900}"
POLL_SEC="${POLL_SEC:-10}"

json_count() {
  local url="$1"
  local jq_expr="$2"
  curl -sf "${url}" | python3 -c "import json,sys; d=json.load(sys.stdin); print(${jq_expr})"
}

wait_for_main_server() {
  local deadline=$((SECONDS + TIMEOUT_SEC))
  until curl -sf "${BASE_URL}/api/overview" >/dev/null 2>&1; do
    if [ "${SECONDS}" -ge "${deadline}" ]; then
      echo "main-server not reachable at ${BASE_URL} within ${TIMEOUT_SEC}s" >&2
      return 1
    fi
    echo "waiting for main-server..."
    sleep "${POLL_SEC}"
  done
}

wait_for_counts() {
  local deadline=$((SECONDS + TIMEOUT_SEC))
  local hosts collectors runs
  while true; do
    hosts="$(json_count "${BASE_URL}/api/hosts" "len(d.get('rows', []))" 2>/dev/null || echo 0)"
    collectors="$(json_count "${BASE_URL}/api/collector/nodes" "len(d.get('nodes', []))" 2>/dev/null || echo 0)"
    runs="$(json_count "${BASE_URL}/api/runs?limit=500" "len(d.get('runs', []))" 2>/dev/null || echo 0)"
    echo "progress: hosts=${hosts}/${EXPECTED_COLLECTORS} collectors=${collectors}/${EXPECTED_COLLECTORS} runs=${runs}"
    if [ "${hosts}" -ge "${EXPECTED_COLLECTORS}" ] && [ "${collectors}" -ge "${EXPECTED_COLLECTORS}" ] && [ "${runs}" -ge "${EXPECTED_COLLECTORS}" ]; then
      return 0
    fi
    if [ "${SECONDS}" -ge "${deadline}" ]; then
      echo "timed out waiting for ${EXPECTED_COLLECTORS} nodes (hosts=${hosts} collectors=${collectors} runs=${runs})" >&2
      return 1
    fi
    sleep "${POLL_SEC}"
  done
}

echo "E2E verify: ${BASE_URL} expecting ${EXPECTED_COLLECTORS} collector node(s)"
wait_for_main_server
wait_for_counts

overview="$(curl -sf "${BASE_URL}/api/overview")"
summary_servers="$(python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('summary',{}).get('servers',0))" <<<"${overview}")"
if [ "${summary_servers}" -lt "${EXPECTED_COLLECTORS}" ]; then
  echo "overview summary.servers=${summary_servers} want >= ${EXPECTED_COLLECTORS}" >&2
  exit 1
fi

echo ""
echo "PASS: full flow verified"
echo "  overview servers: ${summary_servers}"
echo "  hosts rows:       $(json_count "${BASE_URL}/api/hosts" "len(d.get('rows', []))")"
echo "  collector nodes:  $(json_count "${BASE_URL}/api/collector/nodes" "len(d.get('nodes', []))")"
echo "  scan runs:        $(json_count "${BASE_URL}/api/runs?limit=500" "len(d.get('runs', []))")"
echo "  dashboard:        ${BASE_URL}/"
