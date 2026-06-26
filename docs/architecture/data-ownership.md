# Data Ownership

Each service owns its database. Cross-service relationships use stable identifiers and service APIs, not cross-database JOINs.

## Ownership Matrix

| Data | Owner | Notes |
|---|---|---|
| Organizations | `auth-svc` | Organization is part of identity tenancy. |
| Users | `auth-svc` | User identity, profile, status, password hash, MFA flags. |
| Auth providers | `auth-svc` | Local, LDAP, OIDC provider configuration. |
| User credentials | `auth-svc` | External identity provider bindings. |
| Sessions | `auth-svc` | Active sessions, revoked sessions, device metadata. |
| Refresh tokens | `auth-svc` | Stored as hashes; Redis can accelerate lookups. |
| Roles | `iam-svc` | Authorization model. |
| Permissions | `iam-svc` | Permission tree and permission codes. |
| User-role bindings | `iam-svc` | Stores `user_id`, not user profile details. |
| API permission mapping | `iam-svc` | Maps method/path to permission code. |
| Policies and resources | `iam-svc` | Future ABAC model. |
| Audit logs | `audit-svc` | Append-oriented audit records. |
| Notification channels/templates/logs | `notify-svc` | Notification domain data. |
| Assets and topology | `cmdb-svc` | Phase 2. |
| Metrics and alerts | `monitor-svc` | Phase 3. |
| Tickets and approvals | `ticket-svc` | Phase 4. |
| Releases and pipelines | `deploy-svc` | Phase 5. |
| Scripts and jobs | `automation-svc` | Phase 6. |

## Identity References

Services can store these identifiers:

- `user_id`
- `org_id`
- `created_by`
- `updated_by`
- `resource_owner_id`

When display details are needed, `bff-api` should compose them by calling the owning service. Domain databases should not duplicate user profile fields unless a local immutable snapshot is required for audit or historical display.

## Database Naming

Phase 1 databases:

- `auth_svc`
- `iam_svc`
- `audit_svc`
- `notify_svc`
- `bff_api` is stateless by default and should not need a database in phase 1.

Future service databases:

- `cmdb_svc`
- `monitor_svc`
- `ticket_svc`
- `deploy_svc`
- `automation_svc`

## Cache Ownership

| Cache | Owner | Invalidation |
|---|---|---|
| Session and token cache | `auth-svc` | Logout, token revoke, session expiry. |
| User permission snapshot | `iam-svc` or `bff-api` | `user.role_changed`, `role.permission_changed`, `user.deleted`. |
| Frontend bootstrap/menu cache | `bff-api` | Permission-change events and short TTL. |

## Consistency Rule

For cross-service consistency, prefer events over distributed transactions. For example:

1. `auth-svc` disables a user.
2. `auth-svc` publishes `user.disabled`.
3. `iam-svc` or BFF caches evict permission/session-derived data.
4. `audit-svc` records the event.
