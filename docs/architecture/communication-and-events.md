# Communication and Events

The platform uses three communication styles:

1. External HTTP from client to APISIX.
2. Internal HTTP from APISIX to `bff-api`.
3. gRPC from `bff-api` to domain services.
4. Kafka event streams between platform services.

## Synchronous Calls

Use gRPC for service-to-service calls that need an immediate answer. Phase 1 BFF calls already use generated gRPC clients for the same capability surface:

- `bff-api -> auth-svc`: login, refresh, token verify, current user, user CRUD, organization and system config APIs.
- `bff-api -> iam-svc`: permission checks, roles, permissions, API permission mappings, user-role bindings.
- `bff-api -> audit-svc`: audit log query and event type query.
- `bff-api -> notify-svc`: channel/template/log query and send commands.
- Future BFF calls to CMDB, Monitor, Ticket, Deploy, and Automation should follow the same gRPC pattern.

Domain service HTTP endpoints are kept only for internal compatibility, health checks, or debugging. They are not public client APIs.

Domain service to domain service calls are allowed when the dependency is explicit and stable, but avoid tight chains. If the operation can complete asynchronously, prefer an event.

## Asynchronous Events

Use Kafka for business events, audit events, notification triggers, and cache invalidation.

## Phase 1 Events

| Event | Producer | Consumers | Purpose |
|---|---|---|---|
| `user.created` | `auth-svc` | `audit-svc`, `iam-svc` | Audit and optional authorization bootstrap. |
| `user.updated` | `auth-svc` | `audit-svc` | Audit profile/status changes. |
| `user.deleted` | `auth-svc` | `iam-svc`, `audit-svc` | Cleanup user-role bindings and audit. |
| `user.disabled` | `auth-svc` | `iam-svc`, `bff-api`, `audit-svc` | Evict caches and record status change. |
| `user.login` | `auth-svc` | `audit-svc` | Login audit. |
| `user.login_failed` | `auth-svc` | `audit-svc` | Failed login audit and security analysis. |
| `user.logout` | `auth-svc` | `audit-svc` | Logout audit. |
| `user.role_changed` | `iam-svc` | `bff-api`, `audit-svc` | Permission cache invalidation and audit. |
| `role.created` | `iam-svc` | `audit-svc` | Role audit. |
| `role.updated` | `iam-svc` | `audit-svc` | Role audit. |
| `role.deleted` | `iam-svc` | `audit-svc` | Role audit. |
| `role.permission_changed` | `iam-svc` | `bff-api`, `audit-svc` | Permission cache invalidation and audit. |
| `notification.requested` | Any service | `notify-svc`, `audit-svc` | Send a notification and audit the request. |

## Event Envelope

All events should use a stable envelope:

```json
{
  "event_id": "uuid",
  "event_type": "user.login",
  "org_id": "uuid",
  "actor_user_id": "uuid",
  "resource_type": "auth.user",
  "resource_id": "uuid",
  "trace_id": "request trace id",
  "occurred_at": "2026-06-24T10:30:00+08:00",
  "payload": {}
}
```

## Proto Ownership

Protobuf files live under `pkg/proto/<service>/v1`. Each service owns its public gRPC contract. Generated files are produced by:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/gen-proto.ps1
```

Linux / macOS:

```bash
bash scripts/gen-proto.sh
```

The local toolchain uses:

- `.tools/protoc-35.1/bin/protoc.exe`
- `.gopath/bin/protoc-gen-go.exe`
- `.gopath/bin/protoc-gen-go-grpc.exe`
