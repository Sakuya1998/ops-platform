# 数据归属

每个服务拥有自己的数据库。跨服务关系使用稳定标识和服务 API，不使用跨库 JOIN。

## 归属矩阵

| 数据 | 归属服务 | 说明 |
|---|---|---|
| 组织 | `auth-svc` | 组织是身份租户的一部分。 |
| 用户 | `auth-svc` | 用户身份、资料、状态、密码哈希、MFA 标记。 |
| 认证提供商 | `auth-svc` | Local、LDAP、OIDC 提供商配置。 |
| 用户凭证绑定 | `auth-svc` | 外部身份提供商绑定关系。 |
| 会话 | `auth-svc` | 活跃会话、已吊销会话、设备元数据。 |
| Refresh Token | `auth-svc` | 以哈希形式存储，Redis 可加速查询。 |
| 角色 | `iam-svc` | 授权模型。 |
| 权限 | `iam-svc` | 权限树和权限码。 |
| 用户角色绑定 | `iam-svc` | 只保存 `user_id`，不保存用户详情。 |
| API 权限映射 | `iam-svc` | 将 method/path 映射到权限码。 |
| 策略和资源 | `iam-svc` | 未来 ABAC 模型。 |
| 审计日志 | `audit-svc` | 追加式审计记录。 |
| 通知渠道/模板/日志 | `notify-svc` | 通知领域数据。 |
| 资产和拓扑 | `cmdb-svc` | 二期。 |
| 指标和告警 | `monitor-svc` | 三期。 |
| 工单和审批 | `ticket-svc` | 四期。 |
| 发布和流水线 | `deploy-svc` | 五期。 |
| 脚本和作业 | `automation-svc` | 六期。 |

## 身份引用

服务可以保存以下标识：

- `user_id`
- `org_id`
- `created_by`
- `updated_by`
- `resource_owner_id`

需要展示用户详情时，由 `bff-api` 调用数据归属服务进行组合。领域数据库不应复制用户资料字段，除非审计或历史展示需要不可变快照。

## 数据库命名

一期数据库：

- `auth_svc`
- `iam_svc`
- `audit_svc`
- `notify_svc`
- `bff_api` 默认无状态，一期不需要数据库。

后续服务数据库：

- `cmdb_svc`
- `monitor_svc`
- `ticket_svc`
- `deploy_svc`
- `automation_svc`

## 缓存归属

| 缓存 | 归属服务 | 失效机制 |
|---|---|---|
| 会话和 Token 缓存 | `auth-svc` | 登出、Token 吊销、会话过期。 |
| 用户权限快照 | `iam-svc` 或 `bff-api` | `user.role_changed`、`role.permission_changed`、`user.deleted`。 |
| 前端初始化/菜单缓存 | `bff-api` | 权限变更事件和短 TTL。 |

## 一致性规则

跨服务一致性优先使用事件，不使用分布式事务。例如：

1. `auth-svc` 禁用用户。
2. `auth-svc` 发布 `user.disabled`。
3. `iam-svc` 或 BFF 清理权限/会话派生缓存。
4. `audit-svc` 记录该事件。
