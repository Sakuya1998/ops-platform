#!/usr/bin/env pwsh
# Initialize databases for all services
param(
    [string]$PGHost = "localhost",
    [int]$PGPort = 5432,
    [string]$PGUser = "opsadmin",
    [string]$PGPassword = "ops@2026"
)

$databases = @("auth_svc", "iam_svc", "audit_svc", "notify_svc")
$env:PGPASSWORD = $PGPassword

foreach ($db in $databases) {
    Write-Output "Creating database: $db"
    psql -h $PGHost -p $PGPort -U $PGUser -d postgres -c "CREATE DATABASE $db;" 2>$null
    Write-Output "Database $db ready"
}

# Run migrations
$migrations = @(
    @{ svc = "auth-svc"; db = "auth_svc" }
    @{ svc = "iam-svc"; db = "iam_svc" }
    @{ svc = "audit-svc"; db = "audit_svc" }
    @{ svc = "notify-svc"; db = "notify_svc" }
)

foreach ($m in $migrations) {
    $migrationFile = "services/$($m.svc)/migrations/001_init.sql"
    if (Test-Path $migrationFile) {
        Write-Output "Running migration for $($m.svc)..."
        Get-Content $migrationFile | psql -h $PGHost -p $PGPort -U $PGUser -d $m.db
    }
}

Write-Output "Database initialization complete!"
