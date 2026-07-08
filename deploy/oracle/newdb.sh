#!/usr/bin/env bash
# Provision an isolated database + user in the shared edge-postgres.
# Usage:  bash deploy/oracle/newdb.sh <name> <password>
#   e.g.  bash deploy/oracle/newdb.sh kinnect s3cr3t
# Idempotent-ish: safe to re-run; ignores "already exists" errors.
set -euo pipefail
NAME="${1:?usage: newdb.sh <name> <password>}"
PASS="${2:?usage: newdb.sh <name> <password>}"

docker exec -i edge-postgres psql -U postgres <<SQL || true
CREATE USER ${NAME} WITH PASSWORD '${PASS}';
CREATE DATABASE ${NAME} OWNER ${NAME};
GRANT ALL PRIVILEGES ON DATABASE ${NAME} TO ${NAME};
SQL
echo "==> database '${NAME}' ready (user '${NAME}')"
