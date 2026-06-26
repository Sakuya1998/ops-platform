#!/usr/bin/env bash
set -euo pipefail

PG_HOST="localhost"
PG_PORT="5432"
PG_USER="opsadmin"
PG_PASSWORD="ops@2026"

usage() {
  cat <<'EOF'
Usage: scripts/init-db.sh [options]

Options:
  --pg-host HOST            PostgreSQL host, default: localhost
  --pg-port PORT            PostgreSQL port, default: 5432
  --pg-user USER            PostgreSQL user, default: opsadmin
  --pg-password PASSWORD    PostgreSQL password, default: ops@2026
  -h, --help                Show help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pg-host)
      PG_HOST="$2"
      shift 2
      ;;
    --pg-port)
      PG_PORT="$2"
      shift 2
      ;;
    --pg-user)
      PG_USER="$2"
      shift 2
      ;;
    --pg-password)
      PG_PASSWORD="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Required command '$1' was not found in PATH" >&2
    exit 1
  fi
}

run_psql() {
  local db="$1"
  shift
  PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$db" "$@"
}

require_command psql

databases=(auth_svc iam_svc audit_svc notify_svc)
for db in "${databases[@]}"; do
  echo "Creating database: $db"
  run_psql postgres -c "CREATE DATABASE $db;" 2>/dev/null || true
  echo "Database $db ready"
done

declare -A migrations=(
  ["auth-svc"]="auth_svc"
  ["iam-svc"]="iam_svc"
  ["audit-svc"]="audit_svc"
  ["notify-svc"]="notify_svc"
)

for svc in auth-svc iam-svc audit-svc notify-svc; do
  db="${migrations[$svc]}"
  migration_file="services/${svc}/migrations/001_init.sql"
  if [[ -f "$migration_file" ]]; then
    echo "Running migration for $svc..."
    run_psql "$db" -v ON_ERROR_STOP=1 < "$migration_file"
  fi
done

echo "Database initialization complete!"
