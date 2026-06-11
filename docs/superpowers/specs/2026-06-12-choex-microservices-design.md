# ChoexManager 微服务化架构设计

> 设计日期：2026-06-12
> 状态：已确认，待实现

## 1. 背景与动机

当前 ChoexManager 是一个 Go 单体应用，所有功能模块（Auth、Calendar、Bills、Passwords、Memories、Agent、Proxy）运行在同一进程中，共享单库 `choex_manager`。此次改造目标是将全部应用拆分为独立微服务，对齐字节跳动技术栈（gRPC + Protobuf），获得更好的隔离性、独立部署能力和扩展性。

## 2. 技术选型

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 服务通信 | **gRPC** + Protobuf | 强类型契约，高性能，字节跳动主流方案 |
| 数据库 | **每服务独立数据库** | 纯粹微服务模式，各自独立部署 |
| 代码仓库 | **Multi-Repo**（每服务独立 Git 仓库） | 独立 CI/CD，独立版本管理 |
| 前端对接 | **API Gateway（BFF 模式）** | 前端只调 Gateway，内部 gRPC 转发，前端无感知 |
| Agent Tool Calling | **gRPC 直连**，非 MCP | Agent 直接调各服务 gRPC 接口，无需中间协议层 |

## 3. 服务拆分

### 3.1 服务清单（共 8 个仓库）

| 仓库名 | 角色 | gRPC 端口 | 独立数据库 | 说明 |
|--------|------|-----------|-----------|------|
| `choex-proto` | Proto 定义中心 | - | - | 所有 `.proto` 文件 + 代码生成，其他仓库通过 `go mod` 引用 |
| `choex-gateway` | API 网关 | 8080 (HTTP) | - | 前端唯一入口，JWT 认证，HTTP→gRPC 转发，Proxy 路由 |
| `choex-auth` | 认证服务 | 9001 | `choex_auth` | 注册、登录、Token 签发/验证 |
| `choex-calendar` | 日程服务 | 9002 | `choex_calendar` | 日程 CRUD |
| `choex-bill` | 记账服务 | 9003 | `choex_bill` | 账单 CRUD + 月度统计 |
| `choex-password` | 密码管理 | 9004 | `choex_password` | 密码 AES-256-GCM 加密存储与查询 |
| `choex-memory` | Agent 记忆 | 9005 | `choex_memory` | 记忆管理 + 版本控制 |
| `choex-agent` | AI 对话 | 9006 | `choex_agent` | DeepSeek 对话 + gRPC Tool Calling |

### 3.2 不在拆分范围的

- **`choex-web`（前端）**：保持独立仓库，唯一改动是将请求目标改为 Gateway
- **LLM 客户端库**：内化到 `choex-agent` 服务中（原为独立 `llm/` 包）
- **Proxy（浏览器代理）**：保留在 Gateway 中作为一个 HTTP 路由，非独立服务

## 4. 通信架构

### 4.1 整体拓扑

```
React SPA :5173
    │  HTTP/REST (JWT Header)
    ▼
Gateway :8080
    │  gRPC (Metadata 携带 user_id)
    ├──▶ Auth :9001      ── DB: choex_auth
    ├──▶ Calendar :9002  ── DB: choex_calendar
    ├──▶ Bill :9003      ── DB: choex_bill
    ├──▶ Password :9004  ── DB: choex_password
    ├──▶ Memory :9005    ── DB: choex_memory
    └──▶ Agent :9006     ── DB: choex_agent
              │  Agent 内部通过 gRPC 调用其他服务
              ├──▶ Calendar
              ├──▶ Bill
              ├──▶ Password
              └──▶ Memory
```

### 4.2 认证鉴权流程

1. **注册/登录**：前端 → Gateway HTTP → gRPC → Auth Service，返回 JWT
2. **后续请求**：前端携带 `Authorization: Bearer <JWT>` → Gateway 解析 JWT → 将 `user_id` 注入 gRPC Metadata → 转发到各业务服务
3. **数据隔离**：各业务服务从 gRPC Metadata 读取 `user_id`，查询时强制过滤 `WHERE user_id = ?`

关键设计：**认证下沉到 Gateway 一层**，各业务服务不做 JWT 验证，只从 metadata 读取 `user_id`。

### 4.3 Agent Tool Calling 流程

使用 DeepSeek 原生 Function Calling API（非 prompt 解析），避免上次的误判问题：

```
1. 用户输入 "帮我记一笔账，午餐 30 元"
2. Agent Service 构造请求发给 DeepSeek（附带工具定义 JSON Schema）
3. DeepSeek 返回 function_call: create_bill(amount=30, type=expense, category="餐饮")
4. Agent Service 通过 gRPC 调用 Bill Service.CreateBill()
5. 将工具结果发给 DeepSeek 生成自然语言回复
6. 流式返回给前端
```

Agent 可用的工具注册表：

| 工具名 | 目标服务 | 功能 |
|--------|---------|------|
| `query_calendar` | Calendar Service | 查询/创建/修改日程 |
| `manage_bills` | Bill Service | 记账/查询/统计 |
| `search_password` | Password Service | 搜索密码记录 |
| `get_memories` | Memory Service | 获取激活的 Agent 记忆 |

## 5. 数据库设计

### 5.1 拆分策略

- 共享 MySQL 实例，独立 database（`choex_auth`, `choex_calendar`, ...）
- 用户注册时仅在 `choex_auth.users` 创建记录，其他服务按需创建数据
- `user_id` 在各服务中作为普通字段存储，不做跨库外键约束
- 数据隔离由各服务的 `WHERE user_id = ?` 查询保证

### 5.2 各服务独立数据库表结构

**choex_auth**:
- `users` (id, username, email, password_hash, created_at, updated_at)

**choex_calendar**:
- `events` (id, user_id, title, description, location, start_time, end_time, all_day, color, created_at, updated_at)

**choex_bill**:
- `bills` (id, user_id, amount, type, category, note, bill_date, created_at, updated_at)

**choex_password**:
- `passwords` (id, user_id, title, url, username, encrypted_password, note, category, created_at, updated_at)

**choex_memory**:
- `agent_memories` (id, user_id, name, icon, is_active, created_at, updated_at)
- `memory_versions` (id, memory_id, version_type, content, created_at)

**choex_agent**:
- `chat_history` (id, user_id, role, content, created_at) — 新增会话记录表

### 5.3 数据迁移

采用**代码驱动迁移**策略：各服务启动时通过 GORM `AutoMigrate` 自动建表，不编写 SQL 迁移脚本。原有单体 `choex_manager` 数据库保留不动，各服务从空库启动。

## 6. 仓库结构

每个服务仓库遵循统一的 Go 项目标准结构：

```
choex-<service>/
├── go.mod                    # 独立的 Go module，通过 go mod 引用 choex-proto
├── go.sum
├── cmd/
│   └── server/
│       └── main.go           # 入口：读取配置 → 连接DB → 注册gRPC服务 → 启动
├── internal/
│   ├── config/
│   │   └── config.go         # 环境变量配置（DB连接串、gRPC端口等）
│   ├── database/
│   │   └── database.go       # MySQL 连接初始化 + AutoMigrate
│   ├── model/
│   │   └── *.go              # GORM 模型定义
│   ├── service/
│   │   └── *.go              # 业务逻辑层
│   └── server/
│       └── grpc.go           # gRPC 服务注册 + 接口实现
├── Dockerfile                # 多阶段构建，最终镜像约 15MB
└── README.md
```

**Proto 引用方式**：各服务仓库使用 Git Submodule 引入 `choex-proto`，在 `go.mod` 中通过 `replace` 指令指向本地路径。

## 7. 部署方案

### 7.1 本地开发（docker-compose）

```yaml
services:
  mysql:
    image: mysql:9.6
    ports: ["3306:3306"]
    # 初始化脚本创建 6 个数据库

  redis:
    image: redis:8.8
    ports: ["6379:6379"]

  gateway:
    build: ../choex-gateway
    ports: ["8080:8080"]

  auth:
    build: ../choex-auth
    # :9001

  calendar:
    build: ../choex-calendar
    # :9002

  bill:
    build: ../choex-bill
    # :9003

  password:
    build: ../choex-password
    # :9004

  memory:
    build: ../choex-memory
    # :9005

  agent:
    build: ../choex-agent
    # :9006
```

### 7.2 环境变量

各服务通过环境变量配置，不依赖配置文件：

| 变量 | 说明 | 示例 |
|------|------|------|
| `GRPC_PORT` | gRPC 监听端口 | `9001` |
| `DB_HOST` | MySQL 地址 | `localhost` |
| `DB_PORT` | MySQL 端口 | `3306` |
| `DB_USER` | 数据库用户 | `root` |
| `DB_PASSWORD` | 数据库密码 | `choex2025` |
| `DB_NAME` | 数据库名 | `choex_auth` |
| `JWT_SECRET` | JWT 签名密钥 | (仅 Auth 和 Gateway 需要) |
| `ENCRYPTION_KEY` | AES 加密密钥 | (仅 Password Service 需要) |
| `DEEPSEEK_API_KEY` | DeepSeek API Key | (仅 Agent Service 需要) |

## 8. 前端改动

前端 `choex-web` 几乎不需要改代码：

- API 地址从 `http://localhost:8080` 改为 `http://localhost:8080`（不变，Gateway 接管原端口）
- Agent SSE 连接保持 `/api/agent/chat`，由 Gateway 转发到 Agent Service
- 所有 RESTful 接口路径维持不变

唯一需要在 Vite 开发服务器配置的：`proxy` 指向 Gateway `:8080`，同当前配置。

## 9. 实现顺序

建议按照以下顺序分阶段实现，确保每一阶段都可验证：

**Phase 1：基础设施**
1. 创建 `choex-proto` 仓库，编写所有 `.proto` 文件
2. 实现 `choex-gateway`（HTTP 路由 + JWT + gRPC 客户端代理）
3. 实现 `choex-auth`（注册/登录/Token验证）

**Phase 2：核心业务服务**
4. 实现 `choex-calendar`
5. 实现 `choex-bill`
6. 实现 `choex-password`

**Phase 3：智能体服务**
7. 实现 `choex-memory`
8. 实现 `choex-agent`（对话 + Tool Calling）

**Phase 4：整合验证**
9. 编写 `docker-compose.yml` 一键启动脚本
10. 端到端测试所有功能

## 10. 风险与注意事项

1. **gRPC Server-Side Streaming**：Agent SSE 需要在 Gateway 层做 gRPC Streaming → SSE 转换，这是技术复杂点
2. **Multi-Repo 维护成本**：8 个仓库意味着 8 个 go.mod、8 个 Dockerfile，proto 变更需同步多个仓库的子模块
3. **事务边界**：原单体中跨表事务（如 Memory 的 Create 需要同时建 memory + version）保留在服务内，跨服务场景不需要分布式事务
4. **Tool Calling 可靠性**：必须使用 DeepSeek 原生 Function Calling API，不能退回 prompt 解析模式
