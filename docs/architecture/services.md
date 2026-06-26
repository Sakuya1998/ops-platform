# Service Catalog and Boundaries

This page defines the service catalog for phase 1 and later phases. It also defines which service owns each business capability.

## Service Catalog

| Service | Phase | Public to client | Main protocol | Responsibility |
|---|---:|---|---|---|
| `apisix` | 1 | Yes | HTTP / HTTPS | Outermost edge gateway, route entry, TLS, CORS, rate limit, traffic governance, observability hooks. |
| `bff-api` | 1 | Via APISIX | HTTP to client, gRPC to services | Frontend API layer, DTO composition, page-oriented APIs, permission orchestration, response shaping. |
| `auth-svc` | 1 | No | gRPC, internal HTTP compatibility | Identity center: organizations, users, credentials, sessions, refresh tokens, LDAP/OIDC, MFA, login security. |
| `iam-svc` | 1 | No | gRPC, internal HTTP compatibility | Authorization center: roles, permissions, user-role bindings, API permission mapping, RBAC checks, future ABAC policy. |
| `audit-svc` | 1 | No | gRPC, Kafka consumer, internal HTTP compatibility | Audit event ingestion and audit log query. |
| `notify-svc` | 1 | No | gRPC, Kafka consumer, internal HTTP compatibility | Notification channels, templates, send logs, notification dispatch. |
| `cmdb-svc` | 2 | No | gRPC, Kafka producer | Assets, resource models, lifecycle, relationship topology. |
| `monitor-svc` | 3 | No | gRPC, Kafka producer | Metrics, alert rules, alert events, TSDB integration. |
| `ticket-svc` | 4 | No | gRPC, Kafka producer/consumer | Tickets, templates, approvals, SLA, workflow state. |
| `deploy-svc` | 5 | No | gRPC, Kafka producer | Applications, environments, release orders, deployment pipeline orchestration. |
| `automation-svc` | 6 | No | gRPC, Kafka producer/consumer | Scripts, jobs, batch execution, scheduled tasks. |

## Boundary Rules

### APISIX

APISIX is the only component exposed to the client. It must not contain business orchestration logic. Its responsibilities are edge concerns:

- Route external traffic.
- Enforce TLS, CORS, rate limits, and basic request protection.
- Optionally perform lightweight token pre-checks at the edge when needed; RBAC decisions stay in `bff-api` + `iam-svc`.
- Forward client API traffic to `bff-api`.
- Expose gateway metrics.

### bff-api

`bff-api` owns the HTTP API contract consumed by the React SPA. It should not own domain data. It composes domain capabilities for UI workflows:

- Login and session endpoints that delegate to `auth-svc`.
- Current user profile, menu, and permission-aware UI bootstrap APIs.
- User, role, audit, notification, and future module page APIs.
- API-level authorization orchestration through `iam-svc`.
- Explicit typed gRPC clients for each Phase 1 service instead of transparent HTTP proxying.
- Mapping service errors to stable frontend response formats.

### auth-svc

`auth-svc` is the identity source of truth. It owns:

- `organizations`
- `users`
- `user_credentials`
- `auth_providers`
- `sessions`
- `refresh_tokens`
- MFA and password policy data

It must not own roles, permissions, or authorization policies.

### iam-svc

`iam-svc` is the authorization source of truth. It owns:

- `roles`
- `permissions`
- `user_roles`
- `role_permissions`
- `api_permissions`
- `resources`, `policies`, `policy_bindings` for future ABAC

It references `user_id` and `org_id` from `auth-svc`, but does not store user profile details and does not JOIN the Auth database.

### Domain Services

Domain services own their module data and publish business events. They should trust user context only when it comes from BFF or internal service calls. For sensitive resource operations, they can ask IAM for resource-level authorization.

Domain service HTTP endpoints are kept only for internal compatibility, health checks, or debugging. They must not be public client APIs.
