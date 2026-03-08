# Skill Hub

Skill Hub 是一个前后端分离的资源平台，当前支持四类内容：

- `skill`：可复用技能/自动化能力
- `rules`：规则、规范、策略类文本资源
- `mcp`：MCP 服务或元数据资源
- `tools`：工具包、CLI、插件等资源

当前主线能力已经包括：

- `skill / rules` 的 AI 审核、人工复核、重试与 revision 流程
- `mcp / tools` 的自动通过、自动发布上传流
- 点赞、收藏、下载统计
- 用户资料页的上传分页、收藏列表、活动摘要
- PostgreSQL + migration-first 数据库管理
- `/health` 健康检查、CORS allowlist、安全响应头、速率限制
- 非 root 容器运行与结构化日志

## 快速入口

第一次接手项目，建议按这个顺序看：

1. [README.md](./README.md)
2. [DEVELOPMENT.md](./DEVELOPMENT.md)
3. [ARCHITECTURE.md](./ARCHITECTURE.md)
4. [DEPLOYMENT.md](./DEPLOYMENT.md)
5. [db/SCHEMA.md](./db/SCHEMA.md)
6. [ai-review流程.md](./ai-review流程.md)

## 快速启动

### 前置要求

本地开发建议准备：

- Docker Desktop / Docker Engine
- Go `1.25+`
- Node.js `20+`
- npm `10+`

### 1. 启动方式选择

如果你走 `Docker Compose` 完整本地栈：

- 不需要先创建 `backend/.env.local`
- 默认值已经内置在 [docker-compose.yml](./docker-compose.yml)
- 如果你要接入自己的 OpenAI / GitHub token，再通过 shell 环境变量覆盖即可

如果你走“宿主机直接跑 backend/frontend”的开发流，再准备本地 env 文件。

### 2. 宿主机开发时准备环境文件

```bash
cp backend/.env.local.example backend/.env.local
cp frontend/.env.local.example frontend/.env.local
```

本地最关键的变量：

- [backend/.env.local.example](./backend/.env.local.example)
  - `DATABASE_URL`
  - `JWT_SECRET`
- [frontend/.env.local.example](./frontend/.env.local.example)
  - `VITE_API_BASE_URL`

### 3. 一键启动完整本地栈

```bash
docker compose up -d --build

# 关闭
docker compose down

# 清空并关闭
docker compose down -v
```

这会启动：

- PostgreSQL
- Redis
- migration 一次性任务
- backend
- frontend

本地默认访问地址：

- 前端：`http://localhost:5173`
- 后端 API：`http://localhost:8080/api/...`
- 健康检查：`http://localhost:8080/health`

### 4. 宿主机开发流

如果你想直接在本机跑 backend/frontend：

```bash
./scripts/db-local.sh
./scripts/run-all-migrations.sh
cd backend && go run ./cmd/server/main.go
cd frontend && npm ci && npm run dev
```

完整开发说明见 [DEVELOPMENT.md](./DEVELOPMENT.md)。

## 当前项目事实

- 业务 schema 只来自 [db/migrations](./db/migrations)
- backend 启动时不会自动建表或改表
- `skill / rules` 走 reviewed flow；`mcp / tools` 走 auto-published flow
- profile 上传列表已经改成服务端分页接口 `/api/me/uploads`
- 非本地环境必须显式提供安全 `DATABASE_URL` 和 `JWT_SECRET`
- frontend 请求层已经统一处理 401 会话失效

## 常用验证

```bash
cd backend && go test ./...
cd frontend && npm run build
```

前端轻量回归测试也可直接跑：

```bash
cd frontend && npm run test:node
```

更多脚本、测试和本地维护命令见 [DEVELOPMENT.md](./DEVELOPMENT.md)。

## 文档地图

- 开发指南：[DEVELOPMENT.md](./DEVELOPMENT.md)
- 架构总览：[ARCHITECTURE.md](./ARCHITECTURE.md)
- 部署手册：[DEPLOYMENT.md](./DEPLOYMENT.md)
- AI 审核流程：[ai-review流程.md](./ai-review流程.md)
- 数据库说明：[db/SCHEMA.md](./db/SCHEMA.md)
- GitHub Actions：[GITHUB_ACTIONS.md](./GITHUB_ACTIONS.md)
- GitHub 同步配置：[GITHUB_SYNC_SETUP.md](./GITHUB_SYNC_SETUP.md)
- CI/CD 模板：[CI_CD_TEMPLATE.md](./CI_CD_TEMPLATE.md)

## 部署原则

共享环境和生产环境统一遵循：

```text
migrate -> backend -> frontend
```

详细步骤和安全约束见 [DEPLOYMENT.md](./DEPLOYMENT.md)。
