# Skill Hub

AI 驱动的技能共享平台（React + Go），支持 Skill/MCP/Rules/Tools 上传、AI 审核、收藏点赞、下载统计，以及 Skill 到 GitHub 仓库同步。

## 开发模式

当前仓库统一采用 PostgreSQL，并使用版本化 SQL migration 管理表结构。

关键原则：

- 后端启动时不自动改表
- 本地先起数据库，再执行 migration
- 共享环境和生产环境部署时先跑 migration，再发布 backend，再发布 frontend

## 本地快速开始

### 1. 准备环境文件

```bash
cp backend/.env.local.example backend/.env.local
cp frontend/.env.local.example frontend/.env.local
```

至少确认这些变量：

- `backend/.env.local`
  - `DATABASE_URL`
  - `JWT_SECRET`
- `frontend/.env.local`
  - `VITE_API_BASE_URL`

### 2. 启动本地数据库基础设施

```bash
./scripts/db-local.sh
```

这会启动：

- PostgreSQL
- Redis

本地基础设施定义在：

- `infra/docker/compose.local.yml`

### 3. 执行全量 migration

```bash
./scripts/run-all-migrations.sh
```

### 4. 可选：灌本地测试数据

```bash
./scripts/seed-local.sh
```

### 5. 启动后端和前端

后端：

```bash
cd backend
go run cmd/server/main.go
```

前端：

```bash
cd frontend
npm install
npm run dev
```

访问地址：

- 前端（Vite）：`http://localhost:5173`
- 后端 API：`http://localhost:8080/api/...`

## 测试说明

backend 测试现在统一跑在 PostgreSQL 上。

执行前提：

- 本地 PostgreSQL 已启动
- migration 已执行

推荐顺序：

```bash
./scripts/db-local.sh
./scripts/run-all-migrations.sh
cd backend && go test ./...
```

说明：

- 不需要先启动 `go run cmd/server/main.go`
- 不需要先启动 `npm run dev`
- 测试只依赖可访问的 PostgreSQL，不依赖前后端开发服务

## GitHub Actions

仓库内已提供实际可运行的 GitHub Actions 校验流水线：

- `.github/workflows/verify.yml`

它会执行：

- PostgreSQL service + migration + backend tests
- frontend build

详细说明见 [GITHUB_ACTIONS.md](./GITHUB_ACTIONS.md)。

## 本地重置数据

```bash
./scripts/clear-db-data.sh
```

该脚本会：

- 清空 PostgreSQL 中的业务数据
- 清空本地头像、缩略图和上传目录内容

## 迁移规范

所有业务表结构都放在：

- `db/migrations/`

约定：

- `db/init/` 只做本地数据库初始化和扩展启用
- `db/migrations/` 是唯一业务 schema 来源
- `db/seed/` 只放本地或测试用数据

新增 migration 的标准方式：

1. 新建 `db/migrations/NNNN_description.up.sql`
2. 新建 `db/migrations/NNNN_description.down.sql`
3. 本地跑 `./scripts/run-all-migrations.sh`
4. 验证后端测试和前端构建

对于重命名字段、删字段、改类型这类破坏性变化，使用 expand-and-contract：

1. 先加新列
2. 再回填数据
3. 再切换代码读写
4. 最后再删旧列

## 部署流程

共享环境和生产环境统一遵守：

```text
migrate -> backend -> frontend
```

不要依赖：

- 应用启动时自动建表
- `AutoMigrate`
- SQLite 文件数据库

部署前请先准备：

- 外部 PostgreSQL
- 外部或可选 Redis
- backend 运行时环境变量
- frontend 公开环境变量

详细说明见 [DEPLOYMENT.md](./DEPLOYMENT.md)。

## 资源与发布规则

资源路由约定：

- 列表页：`/resource/{skill|rules|mcp|tools}`
- 上传页：`/resource/{type}/upload`
- 详情页：`/resource/{type}/{id}`
- 兼容路由：`/upload`、`/skill/:id`

上传行为：

- `skill`：文件/文件夹上传，AI 审核 + 人工复核
- `rules`：仅 `.md/.txt` 或粘贴 Markdown，AI 审核 + 人工复核
- `mcp`：文章型发布（metadata），无 review，可填 GitHub 链接
- `tools`：文章型发布（metadata/file），无 review，可附压缩包

仅 `resource_type=skill` 会同步 GitHub（`GITHUB_SYNC_ENABLED=true` 时生效）。

## 环境变量

### Backend

| 变量 | 默认值 | 说明 |
|---|---|---|
| `APP_ENV` | `local` | 运行环境标识 |
| `PORT` | `8080` | 后端端口 |
| `DATABASE_URL` | `postgres://skillhub:skillhub@localhost:5432/skillhub_local?sslmode=disable` | PostgreSQL 连接串 |
| `JWT_SECRET` | `change-me-in-production` | JWT 签名密钥 |
| `UPLOAD_DIR` | `./uploads` | 上传目录 |
| `THUMBNAIL_DIR` | `./thumbnails` | 缩略图目录 |
| `OPENAI_API_KEY` | 空 | OpenAI Key |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | OpenAI Base URL |
| `OPENAI_MODEL` | `gpt-4o-mini` | AI 模型 |
| `GITHUB_SYNC_ENABLED` | `false` | 是否开启 Skill GitHub 同步 |
| `GITHUB_TOKEN` | 空 | GitHub PAT（contents 读写权限） |
| `GITHUB_OWNER` | 空 | 目标 owner/org |
| `GITHUB_REPO` | 空 | 目标 repo |
| `GITHUB_BRANCH` | `main` | 同步分支 |
| `GITHUB_BASE_DIR` | `skills` | 仓库根目录 |
| `REDIS_ADDR` | 空 | Redis 地址，例如 `localhost:6379` |
| `REDIS_PASSWORD` | 空 | Redis 密码 |
| `REDIS_DB` | `0` | Redis DB |
| `AI_SKILLS_CACHE_KEY` | `ai:skills_context:v1` | AI 上下文缓存键 |
| `AI_SKILLS_INVALIDATE_CHANNEL` | `ai:skills_context:invalidate` | AI 上下文失效广播 |

### Frontend

| 变量 | 默认值 | 说明 |
|---|---|---|
| `VITE_APP_ENV` | `local` | 前端运行环境标识 |
| `VITE_API_BASE_URL` | `http://localhost:8080` | 后端基础地址，前端会自动补 `/api` |

## 数据去向

默认情况下：

- 结构化业务数据在 PostgreSQL
- 上传文件在后端文件系统目录
- 缩略图在后端文件系统目录
- 头像在后端文件系统目录
- Redis 仅作为可选缓存

如果开启 GitHub 同步：

- `skill` 资源会同步到 GitHub 仓库

如果开启 OpenAI：

- 审核与推荐请求会发送到外部 AI 服务

## 文档

- GitHub 同步配置: [GITHUB_SYNC_SETUP.md](./GITHUB_SYNC_SETUP.md)
- 部署指南（生产环境）: [DEPLOYMENT.md](./DEPLOYMENT.md)
- 架构总览: [ARCHITECTURE.md](./ARCHITECTURE.md)
- GitHub Actions 说明: [GITHUB_ACTIONS.md](./GITHUB_ACTIONS.md)
- CI/CD 模板说明: [CI_CD_TEMPLATE.md](./CI_CD_TEMPLATE.md)
- 设计文档: `docs/plans/2026-03-08-postgresql-migration-architecture-design.md`
- 实施计划: `docs/plans/2026-03-08-postgresql-migration-implementation-plan.md`
