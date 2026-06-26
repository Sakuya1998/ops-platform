#!/usr/bin/env pwsh
<#
.SYNOPSIS
  Build and smoke-test the local Docker Compose deployment through APISIX.
#>
param(
    [string]$ComposeFile = "deploy/docker-compose.yml",
    [string]$APISIXBaseURL = "http://localhost:9080",
    [string]$Username = "admin",
    [string]$Password = "admin@2026"
)

$ErrorActionPreference = "Stop"

function Require-Command {
    param([string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH"
    }
}

function Wait-HttpOK {
    param(
        [string]$URL,
        [int]$TimeoutSeconds = 180
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        try {
            $resp = Invoke-WebRequest -Uri $URL -UseBasicParsing -TimeoutSec 5
            if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 300) {
                return
            }
        } catch {
            Start-Sleep -Seconds 3
        }
    } while ((Get-Date) -lt $deadline)
    throw "Timed out waiting for $URL"
}

Require-Command docker

Write-Output "=== Building and starting Compose stack ==="
docker compose -f $ComposeFile up -d --build

Write-Output "=== Waiting for APISIX health endpoint ==="
Wait-HttpOK "$APISIXBaseURL/health"

Write-Output "=== Seeding default organization, admin user, roles, and permissions ==="
& "$PSScriptRoot/seed.ps1" -UseDockerCompose -ComposeFile $ComposeFile

Write-Output "=== Login through APISIX ==="
$login = Invoke-RestMethod -Method Post -Uri "$APISIXBaseURL/api/v1/auth/login" -ContentType "application/json" -Body (@{
    username = $Username
    password = $Password
    org_code = "default"
    provider = "local"
} | ConvertTo-Json)

$token = $login.data.access_token
if (-not $token) {
    throw "Login response did not contain data.access_token: $($login | ConvertTo-Json -Depth 10)"
}

$headers = @{ Authorization = "Bearer $token" }

Write-Output "=== Verifying core APIs through APISIX ==="
Invoke-RestMethod -Method Get -Uri "$APISIXBaseURL/api/v1/auth/me" -Headers $headers | Out-Null
Invoke-RestMethod -Method Get -Uri "$APISIXBaseURL/api/v1/users" -Headers $headers | Out-Null
Invoke-RestMethod -Method Get -Uri "$APISIXBaseURL/api/v1/roles" -Headers $headers | Out-Null
Invoke-RestMethod -Method Get -Uri "$APISIXBaseURL/api/v1/audit-logs" -Headers $headers | Out-Null

Write-Output "=== Smoke test passed ==="
