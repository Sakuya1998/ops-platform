# 通信与事件

平台使用四类通信方式：

1. 客户端到 APISIX 的外部 HTTP/HTTPS。
2. APISIX 到 `bff-api` 的内部 HTTP。
3. `bff-api` 到领域服务的 gRPC 同步调用。
4. 平台服务之间的 Kafka 异步事件流。

领域服务 HTTP 端点只作为内部兼容、健康检查或调试入口，不作为公开客户端 API。

## 同步调用

需要立即得到结果的服务间调用使用 gRPC。当前一期已经完成：

- `bff-api -> auth-svc`：登录、刷新、Token 校验、当前用户、用户 CRUD、组织和系统配置 API。
- `bff-api -> iam-svc`：权限检查、角色、权限、API 权限映射、用户角色绑定。
- `bff-api -> audit-svc`：审计日志查询和事件类型查询。
- `bff-api -> notify-svc`：渠道、模板、日志查询和发送命令。

后续 BFF 调用 CMDB、监控、工单、发布和自动化服务时同样使用 gRPC。

领域服务之间可以在依赖明确且稳定时同步调用，但要避免过长调用链。如果操作可以异步完成，优先使用事件。

## 异步事件

业务事件、审计事件、通知触发和缓存失效使用 Kafka。

## 一期事件

| 事件 | 生产者 | 消费者 | 目的 |
|---|---|---|---|
| `user.created` | `auth-svc` | `audit-svc`, `iam-svc` | 审计和可选授权初始化。 |
| `user.updated` | `auth-svc` | `audit-svc` | 审计资料/状态变更。 |
| `user.deleted` | `auth-svc` | `iam-svc`, `audit-svc` | 清理用户角色绑定并审计。 |
| `user.disabled` | `auth-svc` | `iam-svc`, `bff-api`, `audit-svc` | 清理缓存并记录状态变更。 |
| `user.login` | `auth-svc` | `audit-svc` | 登录审计。 |
| `user.login_failed` | `auth-svc` | `audit-svc` | 登录失败审计和安全分析。 |
| `user.logout` | `auth-svc` | `audit-svc` | 登出审计。 |
| `user.role_changed` | `iam-svc` | `bff-api`, `audit-svc` | 权限缓存失效和审计。 |
| `role.created` | `iam-svc` | `audit-svc` | 角色审计。 |
| `role.updated` | `iam-svc` | `audit-svc` | 角色审计。 |
| `role.deleted` | `iam-svc` | `audit-svc` | 角色审计。 |
| `role.permission_changed` | `iam-svc` | `bff-api`, `audit-svc` | 权限缓存失效和审计。 |
| `notification.requested` | 任意服务 | `notify-svc`, `audit-svc` | 发送通知并审计请求。 |

## 事件信封

所有事件使用稳定信封：

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

## Proto 归属

Protobuf 文件位于 `pkg/proto/<service>/v1`。每个服务拥有自己的公开 gRPC 契约。生成命令：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/gen-proto.ps1
```

Linux / macOS:

```bash
bash scripts/gen-proto.sh
```

本地工具链：

- `.tools/protoc-35.1/bin/protoc.exe`
- `.gopath/bin/protoc-gen-go.exe`
- `.gopath/bin/protoc-gen-go-grpc.exe`
