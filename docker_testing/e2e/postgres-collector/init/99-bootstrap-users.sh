#!/bin/bash
set -euo pipefail

# Extra roles for CIS/HBA checks (pattern from docker_testing/createuser.sh).
for i in 0 1 2 3 4 5; do
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    DO \$\$
    BEGIN
      IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'user${i}') THEN
        CREATE ROLE user${i} WITH LOGIN PASSWORD 'password' SUPERUSER;
      END IF;
    END
    \$\$;
EOSQL
done

pgbench -i -s 10 -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1 || true

echo "bootstrap users and pgbench schema ready"
