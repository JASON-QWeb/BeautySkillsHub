# Skill Hub

Skill Hub 是一套前后端分离的技能资源平台，支持：

- `skill / rules / mcp / tools` 四类资源发布
- AI 审核与人工复核
- 收藏、点赞、下载统计
- 可选 GitHub 同步
- PostgreSQL + migration-first 数据库管理

当前仓库已经统一切换到 PostgreSQL。数据库结构变更通过版本化 SQL migration 管理，后端启动时不再自动改表。

## 先看这里

如果你第一次接手这个项目，按下面顺序看：

1. [README.md](./README.md)：本地启动、测试、项目入口
2. [ARCHITECTURE.md](./ARCHITECTURE.md)：整体架构、目录边界、数据流
3. [DEPLOYMENT.md](./DEPLOYMENT.md)：如何部署到共享环境或生产环境
4. [GITHUB_ACTIONS.md](./GITHUB_ACTIONS.md)：仓库内 GitHub Actions 校验流水线
5. [CI_CD_TEMPLATE.md](./CI_CD_TEMPLATE.md)：企业内部 CI/CD 扩展模板

## 本地启动

### 前置要求

本地建议准备：

- Docker Desktop / Docker Engine
- Go `1.25+`
- Node.js `20+`
- npm `10+`

### 1. 准备环境文件

```bash
cp backend/.env.local.example backend/.env.local
cp frontend/.env.local.example frontend/.env.local
```

最关键的变量：

- `backend/.env.local`
  - `DATABASE_URL`
  - `JWT_SECRET`

### 2. 方式 A：Docker Compose 一键启动整套本地环境

```bash
docker compose up -d --build
```

这条命令会启动：

- PostgreSQL
- Redis
- migration 一次性任务
- backend
- frontend

查看关键服务日志：

```bash
docker compose logs -f migrate backend frontend
```

停止整套容器：

```bash
docker compose down
```

说明：

- migration 会在 compose 启动时自动执行
- `seed` 不会自动执行，避免每次重启容器都灌演示数据
- 根目录 [docker-compose.yml](./docker-compose.yml) 现在是完整本地栈入口

如果你想手动加入本地 seed，在容器已经启动后执行：

```bash
docker compose exec -T postgres \
  psql -v ON_ERROR_STOP=1 -U skillhub -d skillhub_local < db/seed/local_seed.sql
```

### 3. 方式 B：脚本启动数据库，前后端跑宿主机进程

如果你更希望前后端直接在本机跑，方便调试代码：

#### 3.1 启动本地数据库基础设施

```bash
./scripts/db-local.sh
```

这一步会启动：

- PostgreSQL
- Redis

数据库和缓存定义在：

- `infra/docker/compose.local.yml`

#### 3.2 执行 migration

```bash
./scripts/run-all-migrations.sh
```

#### 3.3 可选：灌本地种子数据

```bash
./scripts/seed-local.sh
```

#### 3.4 启动 backend

```bash
cd backend
go run cmd/server/main.go
```

#### 3.5 启动 frontend

```bash
cd frontend
npm ci
npm run dev
```

### 4. 方式 C：一个 terminal 跑完整套宿主机开发流

如果你想在一个 terminal 里把数据库、迁移、seed、backend、frontend 全部带起来：

```bash
./scripts/dev-all.sh
```

默认会执行：

- `./scripts/db-local.sh`
- `./scripts/run-all-migrations.sh`
- `./scripts/seed-local.sh`
- `go run ./cmd/server/main.go`
- `npm run dev -- --host 0.0.0.0`

如果你不想每次都执行 seed：

```bash
SEED_LOCAL=0 ./scripts/dev-all.sh
```

### 本地访问地址

- 前端：`http://localhost:5173`
- 后端 API：`http://localhost:8080/api/...`
- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`

## 本地日常命令

### 重新跑 migration

```bash
./scripts/run-all-migrations.sh
```

### 重置本地业务数据

```bash
./scripts/clear-db-data.sh
```

这个脚本会：

- 清空 PostgreSQL 里的业务数据
- 清空本地头像、缩略图、上传目录内容

### backend 测试

```bash
./scripts/db-local.sh
./scripts/run-all-migrations.sh
cd backend && go test ./...
```

说明：

- backend 测试现在统一跑在 PostgreSQL 上
- 不需要先启动 `go run cmd/server/main.go`
- 不需要先启动 `npm run dev`

### frontend 构建校验

```bash
cd frontend && npm run build
```

## 项目结构

```text
Skill_Hub/
├── backend/                 Go API、业务逻辑、数据库访问、测试
├── frontend/                React + Vite 前端
├── db/
│   ├── init/                本地 PostgreSQL 初始化脚本
│   ├── migrations/          版本化 SQL migration
│   └── seed/                本地演示/测试数据
├── infra/
│   └── docker/              本地 PostgreSQL / Redis 编排
├── scripts/                 本地数据库、迁移、清理脚本
├── .github/workflows/       GitHub Actions 校验流水线
└── docs/plans/              设计与实施计划文档
```

详细说明见 [ARCHITECTURE.md](./ARCHITECTURE.md)。

## 架构原则

当前仓库遵守这些约束：

- backend 启动时不自动改表
- 所有业务 schema 只来自 `db/migrations/`
- `db/init/` 只做本地数据库初始化，不承担业务建表
- `db/seed/` 只用于本地或测试数据
- 共享环境和生产环境遵循：

```text
migrate -> backend -> frontend
```

## 环境变量

### Backend

后端使用运行时环境变量。关键变量：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `APP_ENV` | `local` | 运行环境标识 |
| `PORT` | `8080` | 后端端口 |
| `DATABASE_URL` | `postgres://skillhub:skillhub@localhost:5432/skillhub_local?sslmode=disable` | PostgreSQL 连接串 |
| `JWT_SECRET` | `change-me-in-production` | JWT 签名密钥 |
| `REDIS_ADDR` | 空 | Redis 地址 |
| `OPENAI_API_KEY` | 空 | OpenAI Key |
| `GITHUB_SYNC_ENABLED` | `false` | 是否开启 GitHub 同步 |

本地参考：

- [backend/.env.local.example](./backend/.env.local.example)

### Frontend

前端只允许公开配置：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `VITE_APP_ENV` | `local` | 前端环境标识 |
| `VITE_API_BASE_URL` | `http://localhost:8080` | 后端基础地址；前端会自动补 `/api` |

本地参考：

- [frontend/.env.local.example](./frontend/.env.local.example)
- [frontend/.env.example](./frontend/.env.example)

## GitHub Actions

仓库内已经有实际可运行的 GitHub Actions workflow：

- [`.github/workflows/verify.yml`](./.github/workflows/verify.yml)

它会执行：

- PostgreSQL service + migration + backend tests
- frontend build

详细说明见 [GITHUB_ACTIONS.md](./GITHUB_ACTIONS.md)。

## 部署摘要

如果你要把它部署到共享环境或生产环境：

1. 准备外部 PostgreSQL
2. 配置 backend 环境变量
3. 先执行 migration
4. 再发布 backend
5. 最后发布 frontend

完整步骤见 [DEPLOYMENT.md](./DEPLOYMENT.md)。

## 数据去向

默认情况下：

- 结构化业务数据存储在 PostgreSQL
- 上传文件存储在后端文件系统目录
- 缩略图存储在后端文件系统目录
- 头像存储在后端文件系统目录
- Redis 仅作为可选缓存

如果开启 GitHub 同步：

- `skill` 资源会同步到 GitHub 仓库

如果开启 OpenAI：

- 审核与推荐请求会发送到外部 AI 服务

## 相关文档

- GitHub 同步配置: [GITHUB_SYNC_SETUP.md](./GITHUB_SYNC_SETUP.md)
- 数据库结构说明: [db/SCHEMA.md](./db/SCHEMA.md)
- 架构总览: [ARCHITECTURE.md](./ARCHITECTURE.md)
- 部署指南: [DEPLOYMENT.md](./DEPLOYMENT.md)
- GitHub Actions: [GITHUB_ACTIONS.md](./GITHUB_ACTIONS.md)
- 企业 CI/CD 模板: [CI_CD_TEMPLATE.md](./CI_CD_TEMPLATE.md)
- PostgreSQL 架构设计: [docs/plans/2026-03-08-postgresql-migration-architecture-design.md](./docs/plans/2026-03-08-postgresql-migration-architecture-design.md)
- PostgreSQL 实施计划: [docs/plans/2026-03-08-postgresql-migration-implementation-plan.md](./docs/plans/2026-03-08-postgresql-migration-implementation-plan.md)
- PostgreSQL 测试统一计划: [docs/plans/2026-03-08-postgresql-test-unification-plan.md](./docs/plans/2026-03-08-postgresql-test-unification-plan.md)
