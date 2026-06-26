# 综合性运维平台 (Operations Platform)

基于微服务架构的企业级运维平台，提供资产管理、监控告警、工单系统、发布部署、自动化运维等核心功能。

## 架构

- **边缘网关**: Apache APISIX (TLS, CORS, limit-req, 路由, 边缘流量治理)
- **客户端 API 层**: bff-api (HTTP API, DTO 聚合, 权限编排)
- **服务发现**: Consul
- **服务间通信**: BFF 到领域服务使用 gRPC (同步) + Kafka (异步事件流)，HTTP 仅保留内部兼容/健康检查入口
- **数据层**: PostgreSQL (按服务拆分) + Redis (缓存)

目标请求链路:

```text
React SPA -> APISIX -> bff-api -> gRPC domain services
```

当前一期实现：

```text
React SPA -> APISIX -> bff-api -> gRPC domain services
```

领域服务 HTTP 端点仅作为内部兼容、健康检查或调试入口保留，不再作为公开客户端入口。

详细架构文档见 [docs/architecture](docs/architecture/README.md)，简体中文版本见 [docs/architecture/zh-CN](docs/architecture/zh-CN/README.md)。

## 服务列表

| 服务 | 端口 | 说明 | 状态 |
|---|---|---|---|
| bff-api | 8070 | 前端 API 聚合层 (HTTP API/DTO/权限编排) | ✅ 一期 |
| auth-svc | 8080 | 身份中心 (组织/用户/凭证/LDAP/OIDC/MFA/Token) | ✅ 一期 |
| iam-svc | 8081 | 授权中心 (RBAC/API 权限/用户角色/策略预留) | ✅ 一期 |
| audit-svc | 8082 | 审计日志服务 | ✅ 一期 |
| notify-svc | 8083 | 通知服务 (邮件/钉钉/企微/飞书) | ✅ 一期 |
| cmdb-svc | 8084 | 资产管理 - CMDB | 📋 二期 |
| monitor-svc | 8085 | 监控告警 | 📋 三期 |
| ticket-svc | 8086 | 工单系统 | 📋 四期 |
| deploy-svc | 8087 | 发布部署 | 📋 五期 |
| automation-svc | 8088 | 自动化运维 | 📋 六期 |

## 快速开始

### 前置条件
- Go 1.22+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL 16
- Redis 7
- Kafka 3.6

### 启动基础设施
```bash
docker-compose -f deploy/docker-compose.yml up -d
```

### 一键 Compose 冒烟测试
```powershell
# Windows / PowerShell
pwsh scripts/smoke-compose.ps1

# Linux / macOS
bash scripts/smoke-compose.sh
```

该脚本会构建并启动 Compose 栈，通过 APISIX 调用 `/health`、登录、当前用户、用户、角色和审计日志 API。

### 初始化数据库
```bash
# Windows / PowerShell
pwsh scripts/init-db.ps1

# Linux / macOS
bash scripts/init-db.sh
```

### 生成 Protobuf 代码
```bash
# Windows / PowerShell
pwsh scripts/gen-proto.ps1

# Linux / macOS
bash scripts/gen-proto.sh
```

### 启动服务 (开发模式)
```bash
# 终端 1: 认证服务
cd services/auth-svc && go run ./cmd/main.go

# 终端 2: 权限服务
cd services/iam-svc && go run ./cmd/main.go

# 终端 3: 审计服务
cd services/audit-svc && go run ./cmd/main.go

# 终端 4: 通知服务
cd services/notify-svc && go run ./cmd/main.go

# 终端 5: BFF API
cd services/bff-api && go run ./cmd/main.go

# 终端 6: 前端
cd web && npm install && npm run dev
```

### 种子数据
```bash
# Windows / PowerShell
pwsh scripts/seed.ps1

# Linux / macOS
bash scripts/seed.sh
```

## 项目结构
```
ops-platform/
├── apisix/              # APISIX 网关配置
├── services/            # 微服务
│   ├── auth-svc/        # 认证
│   ├── iam-svc/         # 授权
│   ├── bff-api/         # 前端 API 聚合
│   ├── audit-svc/       # 审计
│   └── notify-svc/      # 通知
├── pkg/                 # 共享 Go 库
│   ├── consul/          # 服务发现
│   ├── database/        # GORM 配置
│   ├── jwt/             # JWT 工具
│   ├── logger/          # 结构化日志
│   ├── kafka/           # 消息队列
│   ├── trace/           # 链路追踪
│   ├── response/        # 统一响应
│   └── proto/           # Protobuf 定义
├── web/                 # React 前端
├── docs/                # 架构与设计文档
├── deploy/              # 部署配置
└── scripts/             # 工具脚本
```

## 默认账号
- 用户名: `admin`
- 密码: `admin@2026`

