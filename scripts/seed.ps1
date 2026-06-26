#!/usr/bin/env pwsh
<#+
.SYNOPSIS
  Seed initial data: admin user, default roles, permissions, default org.
#>
param(
    [string]$PGHost = "localhost",
    [int]$PGPort = 5432,
    [string]$PGUser = "opsadmin",
    [string]$PGPassword = "ops@2026",
    [switch]$UseDockerCompose,
    [string]$ComposeFile = "deploy/docker-compose.yml"
)

$env:PGPASSWORD = $PGPassword

$ORG_ID = "00000000-0000-0000-0000-000000000001"
$ADMIN_USER_ID = "00000000-0000-0000-0000-000000000010"
$ADMIN_ROLE_ID = "00000000-0000-0000-0000-000000000020"
$ENGINEER_ROLE_ID = "00000000-0000-0000-0000-000000000021"
$VIEWER_ROLE_ID = "00000000-0000-0000-0000-000000000022"
$HASHED_PASSWORD = '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

function Run-SQL {
    param([string]$DB, [string]$SQL)
    if ($UseDockerCompose) {
        $SQL | docker compose -f $ComposeFile exec -T postgres psql -U $PGUser -d $DB -v ON_ERROR_STOP=1
    } else {
        $SQL | psql -h $PGHost -p $PGPort -U $PGUser -d $DB -v ON_ERROR_STOP=1
    }
    if ($LASTEXITCODE -ne 0) { throw "SQL failed on database $DB" }
}

Write-Output "=== Seeding Ops Platform Data ==="

Write-Output "[auth_svc] Creating default organization..."
Run-SQL "auth_svc" @"
INSERT INTO organizations (id, name, code, description, status)
VALUES ('$ORG_ID', 'Default Organization', 'default', 'Default organization created by seed script', 'active')
ON CONFLICT (code) DO NOTHING;
"@

Write-Output "[iam_svc] Creating default roles..."
$roles = @(
    @{ id = $ADMIN_ROLE_ID; name = "System Administrator"; code = "admin"; desc = "Full platform permissions" }
    @{ id = $ENGINEER_ROLE_ID; name = "Ops Engineer"; code = "ops_engineer"; desc = "Operational user" }
    @{ id = $VIEWER_ROLE_ID; name = "Ops Viewer"; code = "ops_viewer"; desc = "Read-only user" }
)
foreach ($r in $roles) {
    Run-SQL "iam_svc" @"
INSERT INTO roles (id, org_id, name, code, description, is_system)
VALUES ('$($r.id)', '$ORG_ID', '$($r.name)', '$($r.code)', '$($r.desc)', true)
ON CONFLICT (org_id, code) DO NOTHING;
"@
}

Write-Output "[iam_svc] Assigning admin role..."
Run-SQL "iam_svc" @"
INSERT INTO user_roles (user_id, role_id)
VALUES ('$ADMIN_USER_ID', '$ADMIN_ROLE_ID')
ON CONFLICT DO NOTHING;
"@

Write-Output "[iam_svc] Creating permissions..."
$perms = @(
    @{ code = "user:create"; name = "User Create"; resource = "user"; action = "create" }
    @{ code = "user:read"; name = "User Read"; resource = "user"; action = "read" }
    @{ code = "user:update"; name = "User Update"; resource = "user"; action = "update" }
    @{ code = "user:delete"; name = "User Delete"; resource = "user"; action = "delete" }
    @{ code = "role:create"; name = "Role Create"; resource = "role"; action = "create" }
    @{ code = "role:read"; name = "Role Read"; resource = "role"; action = "read" }
    @{ code = "role:update"; name = "Role Update"; resource = "role"; action = "update" }
    @{ code = "role:delete"; name = "Role Delete"; resource = "role"; action = "delete" }
    @{ code = "role:assign"; name = "Role Assign"; resource = "role"; action = "assign" }
    @{ code = "org:create"; name = "Organization Create"; resource = "org"; action = "create" }
    @{ code = "org:read"; name = "Organization Read"; resource = "org"; action = "read" }
    @{ code = "org:update"; name = "Organization Update"; resource = "org"; action = "update" }
    @{ code = "audit:read"; name = "Audit Read"; resource = "audit"; action = "read" }
    @{ code = "notify:create"; name = "Notify Create"; resource = "notify"; action = "create" }
    @{ code = "notify:read"; name = "Notify Read"; resource = "notify"; action = "read" }
    @{ code = "notify:update"; name = "Notify Update"; resource = "notify"; action = "update" }
    @{ code = "notify:delete"; name = "Notify Delete"; resource = "notify"; action = "delete" }
)
foreach ($p in $perms) {
    $pid = [guid]::NewGuid().ToString()
    Run-SQL "iam_svc" @"
INSERT INTO permissions (id, code, name, resource, action, type)
VALUES ('$pid', '$($p.code)', '$($p.name)', '$($p.resource)', '$($p.action)', 'api')
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, resource = EXCLUDED.resource, action = EXCLUDED.action;
"@
}

Write-Output "[iam_svc] Granting all permissions to admin role..."
Run-SQL "iam_svc" @"
INSERT INTO role_permissions (role_id, permission_id)
SELECT '$ADMIN_ROLE_ID', id FROM permissions
ON CONFLICT DO NOTHING;
"@

Write-Output "[iam_svc] Granting default permissions to ops engineer role..."
Run-SQL "iam_svc" @"
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
"@

Write-Output "[iam_svc] Granting default permissions to ops viewer role..."
Run-SQL "iam_svc" @"
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
"@

Write-Output "[auth_svc] Creating admin user identity..."
Run-SQL "auth_svc" @"
INSERT INTO users (id, org_id, username, password_hash, display_name, email, status, source)
VALUES ('$ADMIN_USER_ID', '$ORG_ID', 'admin', '$HASHED_PASSWORD', 'System Administrator', 'admin@ops-platform.local', 'active', 'local')
ON CONFLICT (org_id, username) DO UPDATE SET
  password_hash = EXCLUDED.password_hash,
  display_name = EXCLUDED.display_name,
  email = EXCLUDED.email,
  status = EXCLUDED.status,
  source = EXCLUDED.source;
"@

Write-Output "[auth_svc] Creating local auth provider..."
Run-SQL "auth_svc" @"
INSERT INTO auth_providers (id, org_id, provider, name, config, is_enabled)
VALUES (gen_random_uuid(), '$ORG_ID', 'local', 'Local Authentication', '{}', true)
ON CONFLICT DO NOTHING;
"@

Write-Output "=== Seed Complete ==="
Write-Output "Default credentials: admin / admin@2026"
