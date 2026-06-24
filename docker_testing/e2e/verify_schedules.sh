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
EXPECTED_COLLECTORS="${EXPECTED_COLLECTORS:-3}"
EXPECTED_SCHEDULES="${EXPECTED_SCHEDULES:-3}"
TIMEOUT_SEC="${TIMEOUT_SEC:-900}"
POLL_SEC="${POLL_SEC:-10}"

echo "E2E schedule verify: expecting ${EXPECTED_SCHEDULES} cron schedule(s) on ${EXPECTED_COLLECTORS} collector(s)"

"${ROOT}/verify.sh"

deadline=$((SECONDS + TIMEOUT_SEC))
while true; do
  min_jobs="$(curl -sf "${BASE_URL}/api/collector/nodes" | python3 -c "
import json,sys
d=json.load(sys.stdin)
jobs=[int(n.get('scheduled_jobs') or 0) for n in d.get('nodes',[])]
print(min(jobs) if jobs else 0)
")"
  echo "progress: min scheduled_jobs=${min_jobs}/${EXPECTED_SCHEDULES}"
  if [ "${min_jobs}" -ge "${EXPECTED_SCHEDULES}" ]; then
    break
  fi
  if [ "${SECONDS}" -ge "${deadline}" ]; then
    echo "timed out waiting for scheduled_jobs>=${EXPECTED_SCHEDULES} (min=${min_jobs})" >&2
    exit 1
  fi
  sleep "${POLL_SEC}"
done

sample="$(docker ps --filter name=e2e-collector --format '{{.Names}}' | head -1)"
log_ok=false
for c in $(docker ps --filter name=e2e-collector --format '{{.Names}}'); do
  logs="$(docker logs "${c}" 2>&1 || true)"
  if [[ "${logs}" == *"registered ${EXPECTED_SCHEDULES} cron schedule"* ]]; then
    log_ok=true
    sample="${c}"
    break
  fi
done
if [ "${log_ok}" = false ]; then
  echo "collector logs missing 'registered ${EXPECTED_SCHEDULES} cron schedule' (checked all e2e-collector containers)" >&2
  if [ -n "${sample}" ]; then
    docker logs "${sample}" 2>&1 | grep -E 'registered .* cron schedule|WARNING: no cron' | tail -5 >&2 || true
  fi
  exit 1
fi
echo "collector log OK: ${sample} registered ${EXPECTED_SCHEDULES} cron schedule(s)"

echo ""
echo "PASS: separate PII and log parser schedules verified"
echo "  min scheduled_jobs: ${min_jobs}"
echo "  expected schedules: ${EXPECTED_SCHEDULES}"
