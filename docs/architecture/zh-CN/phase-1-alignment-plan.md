# Phase 1 Alignment Status

This file is intentionally written in ASCII to avoid encoding drift on Windows shells.
The detailed Simplified Chinese architecture documents remain under `docs/architecture/zh-CN/`.

## Target Flow

```text
React SPA -> APISIX -> bff-api -> gRPC domain services
```

Current Phase 1 implementation matches the target flow.

Exception: OIDC browser redirect endpoints still use an Auth HTTP fallback from BFF.
Normal Auth, IAM, Audit, and Notify capabilities use gRPC.

## Completed Scope

- `auth-svc` owns identity data: organizations, users, credentials, sessions, tokens, LDAP/OIDC/MFA, and system config.
- `iam-svc` owns authorization data: roles, permissions, user-role bindings, API permission mappings, and future ABAC resources/policies.
- `audit-svc` owns audit log query and audit event persistence.
- `notify-svc` owns notification channels, templates, logs, and send commands.
- `bff-api` is the client HTTP API layer.
- APISIX routes all client API traffic to `bff-api`.
- Transparent HTTP proxying has been removed from BFF.
- BFF uses explicit typed handlers and clients for Auth, IAM, Audit, and Notify.
- BFF can use gateway-injected `X-User-*` context or verify bearer tokens through Auth when gateway context is absent.
- BFF performs endpoint authorization through IAM.
- Phase 1 protobuf contracts exist for Auth, IAM, Audit, and Notify.
- Auth, IAM, Audit, and Notify expose gRPC servers.
- BFF calls Auth, IAM, Audit, and Notify over gRPC.
- Compose exposes APISIX as the client entry. BFF and domain services are internal-only.
- APISIX local Compose mode uses `data_plane + yaml` declarative routes.
- Service Dockerfiles support monorepo build context with `go.work`, `pkg`, and service modules.
- Windows and Linux/macOS scripts exist for `gen-proto`, `init-db`, `seed`, and `smoke-compose`.
- The frontend includes Phase 1 pages: login, dashboard, users, roles, permissions, organizations, audit logs, notifications, system config, and profile.

## Transitional Capabilities

- Domain-service HTTP APIs remain only for internal compatibility, health checks, or debugging.
- OIDC login/callback browser redirect still uses HTTP fallback.
- APISIX handles edge traffic and routing. Page-level authorization stays in BFF + IAM.

## Stage Status

### Stage A: Protobuf Contracts

Status: Complete.

Evidence:

- `pkg/proto/auth/v1/auth.proto`
- `pkg/proto/iam/v1/iam.proto`
- `pkg/proto/audit/v1/audit.proto`
- `pkg/proto/notify/v1/notify.proto`
- `scripts/gen-proto.ps1`
- `scripts/gen-proto.sh`
- `go test ./pkg/...`

### Stage B: Domain gRPC Servers

Status: Complete.

Evidence:

- Auth, IAM, Audit, and Notify gRPC servers are wired into service startup.
- gRPC handlers reuse existing service/repository logic.
- Service tests pass.

### Stage C: BFF gRPC Clients

Status: Complete.

Result:

- Auth gRPC client complete, with OIDC redirect HTTP fallback.
- IAM gRPC client complete.
- Audit gRPC client complete.
- Notify gRPC client complete.

### Stage D: HTTP Exposure Closure

Status: Code and config complete. Runtime smoke test still requires a Docker-capable environment.

Evidence:

- APISIX routes only target `bff-api`.
- Compose does not publish BFF/domain service host ports.
- APISIX, Compose, README, and architecture docs align on the same target flow.

## Verification Commands

Backend and shared packages:

```powershell
$env:GOPATH="$PWD\.gopath"
$env:GOCACHE="$PWD\.tmp-gocache"
$env:GOPROXY="https://proxy.golang.org,direct"
go test ./pkg/... ./services/bff-api/... ./services/auth-svc/... ./services/iam-svc/... ./services/audit-svc/... ./services/notify-svc/...
go build ./pkg/... ./services/bff-api/... ./services/auth-svc/... ./services/iam-svc/... ./services/audit-svc/... ./services/notify-svc/...
```

Frontend:

```powershell
cd web
cmd /c npm run build 2>&1
```

Compose smoke test:

```powershell
pwsh scripts/smoke-compose.ps1
```

Linux/macOS:

```bash
bash scripts/smoke-compose.sh
```

## Current Acceptance Conclusion

- Code, config, scripts, docs, and build-level verification are complete for Phase 1.
- This workstation does not provide Docker, Bash, or an installed WSL distribution, so Compose smoke testing cannot be executed here.
- Final runtime acceptance requires running `scripts/smoke-compose.ps1` or `scripts/smoke-compose.sh` in a Docker-capable environment.
