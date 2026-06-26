# 服务清单与边界

本文定义一期和后续阶段的服务清单，并明确每个业务能力归属哪个服务。

## 服务清单

| 服务 | 阶段 | 是否面向客户端 | 主要协议 | 职责 |
|---|---:|---|---|---|
| `apisix` | 1 | 是 | HTTP / HTTPS | 最外层边缘网关，负责入口路由、TLS、CORS、限流、边缘流量治理和观测入口。 |
| `bff-api` | 1 | 通过 APISIX | 对客户端 HTTP，对服务 gRPC | 前端 API 层，负责 DTO 聚合、面向页面的 API、权限编排和响应整形。 |
| `auth-svc` | 1 | 否 | gRPC，过渡期可保留内部 HTTP | 身份中心：组织、用户、凭证、会话、Refresh Token、LDAP/OIDC、MFA、登录安全。 |
| `iam-svc` | 1 | 否 | gRPC，过渡期可保留内部 HTTP | 授权中心：角色、权限、用户角色绑定、API 权限映射、RBAC 校验、未来 ABAC 策略。 |
| `audit-svc` | 1 | 否 | gRPC，Kafka consumer，过渡期可保留内部 HTTP | 审计事件采集与审计日志查询。 |
| `notify-svc` | 1 | 否 | gRPC，Kafka consumer，过渡期可保留内部 HTTP | 通知渠道、模板、发送日志和通知投递。 |
| `cmdb-svc` | 2 | 否 | gRPC，Kafka producer | 资产、资源模型、生命周期和关系拓扑。 |
| `monitor-svc` | 3 | 否 | gRPC，Kafka producer | 指标、告警规则、告警事件和 TSDB 集成。 |
| `ticket-svc` | 4 | 否 | gRPC，Kafka producer/consumer | 工单、模板、审批、SLA 和流程状态。 |
| `deploy-svc` | 5 | 否 | gRPC，Kafka producer | 应用、环境、发布单和流水线编排。 |
| `automation-svc` | 6 | 否 | gRPC，Kafka producer/consumer | 脚本、作业、批量执行和定时任务。 |

## 边界规则

### APISIX

APISIX 是唯一对客户端暴露的组件，不承载业务编排逻辑。它负责边缘能力：

- 路由外部流量。
- 执行 TLS、CORS、限流和基础请求保护。
- 对受保护的客户端路由执行边缘流量治理。
- 将已认证流量转发给 `bff-api`。
- 暴露网关观测指标。

### bff-api

`bff-api` 拥有 React SPA 消费的 HTTP API 契约，但不拥有领域数据。它为 UI 工作流编排领域能力：

- 登录和会话端点委托给 `auth-svc`。
- 当前用户、菜单、权限感知的页面初始化 API。
- 用户、角色、审计、通知以及后续模块的页面 API。
- 通过 `iam-svc` 做 API 级权限编排。
- 为每个一期服务维护显式类型化 gRPC 客户端，不再使用透明 HTTP 代理。
- 将服务错误映射为稳定的前端响应格式。

### auth-svc

`auth-svc` 是身份数据的唯一事实源。它拥有：

- `organizations`
- `users`
- `user_credentials`
- `auth_providers`
- `sessions`
- `refresh_tokens`
- MFA 和密码策略数据

它不能拥有角色、权限或授权策略。

### iam-svc

`iam-svc` 是授权数据的唯一事实源。它拥有：

- `roles`
- `permissions`
- `user_roles`
- `role_permissions`
- `api_permissions`
- 面向未来 ABAC 的 `resources`、`policies`、`policy_bindings`

它通过 `user_id` 和 `org_id` 引用 `auth-svc` 的身份数据，但不保存用户详情，也不 JOIN Auth 数据库。

### 领域服务

领域服务拥有各自模块的数据并发布业务事件。它们只信任来自 BFF 或内部服务调用的用户上下文。敏感资源操作可以向 IAM 请求资源级授权。

领域服务 HTTP 端点仅作为内部兼容、健康检查或调试入口，不应作为公开客户端 API。

