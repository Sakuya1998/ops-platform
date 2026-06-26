#!/usr/bin/env bash
set -euo pipefail

PG_HOST="localhost"
PG_PORT="5432"
PG_USER="opsadmin"
PG_PASSWORD="ops@2026"
COMPOSE_FILE="deploy/docker-compose.yml"
USE_DOCKER_COMPOSE="false"

usage() {
  cat <<'EOF'
Usage: scripts/seed.sh [options]

Options:
  --pg-host HOST            PostgreSQL host, default: localhost
  --pg-port PORT            PostgreSQL port, default: 5432
  --pg-user USER            PostgreSQL user, default: opsadmin
  --pg-password PASSWORD    PostgreSQL password, default: ops@2026
  --compose-file FILE       Docker Compose file, default: deploy/docker-compose.yml
  --use-docker-compose      Execute psql inside the postgres compose service
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
    --compose-file)
      COMPOSE_FILE="$2"
      shift 2
      ;;
    --use-docker-compose)
      USE_DOCKER_COMPOSE="true"
      shift
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

ORG_ID="00000000-0000-0000-0000-000000000001"
ADMIN_USER_ID="00000000-0000-0000-0000-000000000010"
ADMIN_ROLE_ID="00000000-0000-0000-0000-000000000020"
ENGINEER_ROLE_ID="00000000-0000-0000-0000-000000000021"
VIEWER_ROLE_ID="00000000-0000-0000-0000-000000000022"
HASHED_PASSWORD='$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Required command '$1' was not found in PATH" >&2
    exit 1
  fi
}

run_sql() {
  local db="$1"
  local sql="$2"
  if [[ "$USE_DOCKER_COMPOSE" == "true" ]]; then
    require_command docker
    printf '%s\n' "$sql" | docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U "$PG_USER" -d "$db" -v ON_ERROR_STOP=1
  else
    require_command psql
    PGPASSWORD="$PG_PASSWORD" printf '%s\n' "$sql" | PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$db" -v ON_ERROR_STOP=1
  fi
}

echo "=== Seeding Ops Platform Data ==="

echo "[auth_svc] Creating default organization..."
run_sql "auth_svc" "
INSERT INTO organizations (id, name, code, description, status)
VALUES ('$ORG_ID', 'Default Organization', 'default', 'Default organization created by seed script', 'active')
ON CONFLICT (code) DO NOTHING;
"

echo "[iam_svc] Creating default roles..."
run_sql "iam_svc" "
INSERT INTO roles (id, org_id, name, code, description, is_system)
VALUES
  ('$ADMIN_ROLE_ID', '$ORG_ID', 'System Administrator', 'admin', 'Full platform permissions', true),
  ('$ENGINEER_ROLE_ID', '$ORG_ID', 'Ops Engineer', 'ops_engineer', 'Operational user', true),
  ('$VIEWER_ROLE_ID', '$ORG_ID', 'Ops Viewer', 'ops_viewer', 'Read-only user', true)
ON CONFLICT (org_id, code) DO NOTHING;
"

echo "[iam_svc] Assigning admin role..."
run_sql "iam_svc" "
INSERT INTO user_roles (user_id, role_id)
VALUES ('$ADMIN_USER_ID', '$ADMIN_ROLE_ID')
ON CONFLICT DO NOTHING;
"

echo "[iam_svc] Creating permissions..."
run_sql "iam_svc" "
INSERT INTO permissions (id, code, name, resource, action, type)
VALUES
  (gen_random_uuid(), 'user:create', 'User Create', 'user', 'create', 'api'),
  (gen_random_uuid(), 'user:read', 'User Read', 'user', 'read', 'api'),
  (gen_random_uuid(), 'user:update', 'User Update', 'user', 'update', 'api'),
  (gen_random_uuid(), 'user:delete', 'User Delete', 'user', 'delete', 'api'),
  (gen_random_uuid(), 'role:create', 'Role Create', 'role', 'create', 'api'),
  (gen_random_uuid(), 'role:read', 'Role Read', 'role', 'read', 'api'),
  (gen_random_uuid(), 'role:update', 'Role Update', 'role', 'update', 'api'),
  (gen_random_uuid(), 'role:delete', 'Role Delete', 'role', 'delete', 'api'),
  (gen_random_uuid(), 'role:assign', 'Role Assign', 'role', 'assign', 'api'),
  (gen_random_uuid(), 'org:create', 'Organization Create', 'org', 'create', 'api'),
  (gen_random_uuid(), 'org:read', 'Organization Read', 'org', 'read', 'api'),
  (gen_random_uuid(), 'org:update', 'Organization Update', 'org', 'update', 'api'),
  (gen_random_uuid(), 'audit:read', 'Audit Read', 'audit', 'read', 'api'),
  (gen_random_uuid(), 'notify:create', 'Notify Create', 'notify', 'create', 'api'),
  (gen_random_uuid(), 'notify:read', 'Notify Read', 'notify', 'read', 'api'),
  (gen_random_uuid(), 'notify:update', 'Notify Update', 'notify', 'update', 'api'),
  (gen_random_uuid(), 'notify:delete', 'Notify Delete', 'notify', 'delete', 'api')
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  resource = EXCLUDED.resource,
  action = EXCLUDED.action;
"

echo "[iam_svc] Granting all permissions to admin role..."
run_sql "iam_svc" "
INSERT INTO role_permissions (role_id, permission_id)
SELECT '$ADMIN_ROLE_ID', id FROM permissions
ON CONFLICT DO NOTHING;
"

echo "[iam_svc] Granting default permissions to ops engineer role..."
run_sql "iam_svc" "
INSERT INTO role_permissions (role_id, permission_id)
SELECT '$ENGINEER_ROLE_ID', id FROM permissions
WHERE code IN (
  'user:read',
  'role:read',
  'permission:read',
  'org:read',
  'audit:read',
  'notify:create',
  'notify:read',
  'notify:update'
)
ON CONFLICT DO NOTHING;
"

echo "[iam_svc] Granting default permissions to ops viewer role..."
run_sql "iam_svc" "
INSERT INTO role_permissions (role_id, permission_id)
SELECT '$VIEWER_ROLE_ID', id FROM permissions
WHERE code IN (
  'user:read',
  'role:read',
  'permission:read',
  'org:read',
  'audit:read',
  'notify:read'
)
ON CONFLICT DO NOTHING;
"

echo "[auth_svc] Creating admin user identity..."
run_sql "auth_svc" "
INSERT INTO users (id, org_id, username, password_hash, display_name, email, status, source)
VALUES ('$ADMIN_USER_ID', '$ORG_ID', 'admin', '$HASHED_PASSWORD', 'System Administrator', 'admin@ops-platform.local', 'active', 'local')
ON CONFLICT (org_id, username) DO UPDATE SET
  password_hash = EXCLUDED.password_hash,
  display_name = EXCLUDED.display_name,
  email = EXCLUDED.email,
  status = EXCLUDED.status,
  source = EXCLUDED.source;
"

echo "[auth_svc] Creating local auth provider..."
run_sql "auth_svc" "
INSERT INTO auth_providers (id, org_id, provider, name, config, is_enabled)
VALUES (gen_random_uuid(), '$ORG_ID', 'local', 'Local Authentication', '{}', true)
ON CONFLICT DO NOTHING;
"

echo "=== Seed Complete ==="
echo "Default credentials: admin / admin@2026"
