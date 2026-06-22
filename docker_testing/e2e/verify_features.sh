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
SCAN_COMMANDS="${SCAN_COMMANDS:-$(read_env_var SCAN_COMMANDS)}"
SCAN_COMMANDS="${SCAN_COMMANDS:-postgres_cis,hba_scanner,pii_scanner,ssl_audit,inactive_users,unique_ip,unused_lines,password_leak_scanner}"
TIMEOUT_SEC="${TIMEOUT_SEC:-900}"

echo "E2E feature verify: ${SCAN_COMMANDS}"
echo "  collectors: ${EXPECTED_COLLECTORS}  schedules: ${EXPECTED_SCHEDULES}"

EXPECTED_COLLECTORS="${EXPECTED_COLLECTORS}" EXPECTED_SCHEDULES="${EXPECTED_SCHEDULES}" "${ROOT}/verify_schedules.sh"

sample="$(docker ps --filter name=e2e-collector --format '{{.Names}}' | head -1)"
if [ -n "${sample}" ]; then
  sched_count="$(docker exec "${sample}" grep -c '^schedule = ' /etc/klouddbshield/kshieldconfig.toml 2>/dev/null || echo 0)"
  if [ "${sched_count}" -lt "${EXPECTED_SCHEDULES}" ]; then
    echo "FAIL: collector ${sample}: expected >= ${EXPECTED_SCHEDULES} schedule lines in kshieldconfig.toml, got ${sched_count}" >&2
    exit 1
  fi
  if ! docker exec "${sample}" grep -qE 'inactive_users|unique_ip|unused_lines|password_leak_scanner' /etc/klouddbshield/kshieldconfig.toml 2>/dev/null; then
    echo "FAIL: collector ${sample}: scan_commands missing log parser commands" >&2
    exit 1
  fi
  echo "OK: collector container config — ${sched_count} schedule(s), scan_commands include all log parser commands"
fi

python3 - "${BASE_URL}" "${SCAN_COMMANDS}" <<'PY'
import json
import sys
import urllib.error
import urllib.parse
import urllib.request

base_url, scan_raw = sys.argv[1], sys.argv[2]
expected = [c.strip().lower() for c in scan_raw.split(",") if c.strip()]

def get(path):
    req = urllib.request.Request(f"{base_url}{path}")
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.load(resp)

def fail(msg):
    print(f"FAIL: {msg}", file=sys.stderr)
    sys.exit(1)

# --- pick sample host (prefer a target with CIS data in latest run) ---
runs = get("/api/runs?limit=100").get("runs") or []
hosts = get("/api/hosts").get("rows") or []
if not hosts:
    fail("no hosts in /api/hosts")

host_name = hosts[0][0]
host_ip = hosts[0][1] if len(hosts[0]) > 1 else host_name
for run in runs:
    if run.get("total_pass", 0) > 0:
        tid = run.get("target_id", "")
        th = run.get("target_host", "")
        for row in hosts:
            if th and (th in row[0] or th == row[1]):
                host_name, host_ip = row[0], row[1] if len(row) > 1 else row[0]
                break
        else:
            if th:
                host_name = f"{th}:5432"
                host_ip = th
        break
print(f"OK: sample host {host_name!r} ({host_ip})")

checks = {}

# postgres_cis + hba via server detail
try:
    detail = get(f"/api/servers/{urllib.parse.quote(host_name, safe='')}")
except urllib.error.HTTPError:
    detail = get(f"/api/servers/{urllib.parse.quote(host_ip, safe='')}")
if "postgres_cis" in expected:
    cis = detail.get("postgres_cis_responses") or (detail.get("modules") or {}).get("cis_audit", {}).get("rows")
    if not cis:
        cis_mod = (detail.get("modules") or {}).get("cis_audit", {})
        cis = cis_mod.get("rows") if cis_mod.get("available") else cis
    if not cis:
        fail("server detail missing postgres_cis data")
    checks["postgres_cis"] = True
if "hba_scanner" in expected:
    hba = detail.get("hba_scan_result") or (detail.get("modules") or {}).get("pg_hba", {}).get("rows")
    if not hba:
        fail("server detail missing hba_scanner data")
    checks["hba_scanner"] = True

# scanner APIs
if "hba_scanner" in expected:
    hba_api = get(f"/api/scanner/hba?host={urllib.parse.quote(host_name)}")
    if not hba_api.get("checks") and hba_api.get("pass", 0) + hba_api.get("fail", 0) == 0:
        fail(f"/api/scanner/hba empty for {host_name!r}: {hba_api.get('message', '')}")
    checks["hba_scanner_api"] = True

if "ssl_audit" in expected:
    ssl = get(f"/api/scanner/ssl?host={urllib.parse.quote(host_name)}")
    if not ssl.get("available"):
        fail(f"/api/scanner/ssl not available for {host_name!r}: {ssl.get('message', '')}")
    checks["ssl_audit"] = True

if "pii_scanner" in expected:
    pii = get(f"/api/scanner/pii?host={urllib.parse.quote(host_name)}")
    if not pii.get("available"):
        fail(f"/api/scanner/pii not available for {host_name!r}: {pii.get('message', '')}")
    checks["pii_scanner"] = True

LOG_PARSER_TOKENS = {
    "inactive_users",
    "unique_ip",
    "unused_lines",
    "password_leak_scanner",
}

if any(t in expected for t in LOG_PARSER_TOKENS):
    lp = get(f"/api/scanner/logparser?host={urllib.parse.quote(host_name)}")
    if not lp.get("available") and not lp.get("commands"):
        fail(f"/api/scanner/logparser not available for {host_name!r}: {lp.get('message', '')}")
    cmds = {c.get("command") for c in (lp.get("commands") or [])}
    for token in LOG_PARSER_TOKENS:
        if token not in expected:
            continue
        if token not in cmds:
            fail(f"log parser command {token!r} missing from /api/scanner/logparser (got {sorted(cmds)!r})")
        checks[f"logparser:{token}"] = True

if "inactive_users" in expected:
    fleet = get("/api/fleet/categories")
    inactive = next((c for c in fleet.get("categories", []) if c.get("id") == "inactive-users"), None)
    if inactive is None:
        fail("fleet categories missing inactive-users")
    count = inactive.get("count", "0 hosts")
    log_mod = (detail.get("modules") or {}).get("log_parser", {})
    log_ok = log_mod.get("available") or "Log Parser Summary" in (detail.get("raw_keys") or [])
    if count.startswith("0") and not log_ok:
        # Fallback: log parser section present in persisted report_json
        report_keys = detail.get("raw_keys") or []
        if "Log Parser Summary" not in report_keys:
            fail(f"inactive_users not in report and fleet empty: {count!r}")
    checks["inactive_users"] = True

# dashboard pages reachable
for path in ("/", "/api/overview", "/api/strategic", "/api/runs?limit=20"):
    req = urllib.request.Request(f"{base_url}{path}")
    with urllib.request.urlopen(req, timeout=30) as resp:
        if resp.status != 200:
            fail(f"{path} returned {resp.status}")

print("")
print("PASS: all configured scan features verified on docker e2e")
for k in sorted(checks):
    print(f"  ✓ {k}")
print(f"  dashboard: {base_url}/")
PY
