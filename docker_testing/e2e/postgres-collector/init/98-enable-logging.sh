#!/bin/bash
set -euo pipefail
# Enable CSV logging for logparser e2e (runs once during postgres initdb).
cat >> "${PGDATA}/postgresql.conf" <<'EOF'
logging_collector = on
log_directory = 'log'
log_filename = 'postgresql-%Y-%m-%d.log'
log_connections = on
log_disconnections = on
log_line_prefix = '%t %u %d %h '
EOF
mkdir -p "${PGDATA}/log"
