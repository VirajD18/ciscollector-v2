#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "${ROOT}"

echo "=== E2E full verify (flow + schedules + features) ==="
"${ROOT}/verify_features.sh"
