# Skill Hub Architecture

## Overview

Skill Hub 当前采用前后端分离加独立数据库的结构：

- `frontend/`
  - React + Vite
  - 只消费后端 API
  - 只使用公开环境变量
- `backend/`
  - Go + Gin
  - 负责认证、资源管理、AI 审核、GitHub 同步、缓存集成
  - 通过 `DATABASE_URL` 连接 PostgreSQL
- `db/`
  - `init/`：本地数据库初始化脚本
  - `migrations/`：版本化 SQL schema
  - `seed/`：本地开发种子数据
- `infra/`
  - 本地 Docker 基础设施
  - 当前主要提供 PostgreSQL 和 Redis

## Runtime Topology

### Local

```text
frontend (Vite, :5173)
    -> backend (Go, :8080)
        -> PostgreSQL (:5432)
        -> Redis (:6379, optional cache)
        -> OpenAI API (optional)
        -> GitHub API (optional)
```

### Shared / Production

```text
frontend
    -> backend
        -> external PostgreSQL
        -> optional Redis
        -> optional OpenAI / GitHub integrations
```

## Data Ownership

### PostgreSQL

主业务数据存储在 PostgreSQL：

- `users`
- `skills`
- `skill_likes`
- `skill_favorites`

所有业务 schema 变更必须走：

- `db/migrations/*.up.sql`
- `db/migrations/*.down.sql`

backend 启动时不负责建表，也不负责 schema 修复。

### File Storage

后端本地文件目录仍然存放：

- `uploads/`
- `thumbnails/`
- `avatars/`

这些目录是应用资源，不是主业务数据库。

### Cache

Redis 只做可选缓存，不作为主数据存储。

## Configuration Model

### Backend

backend 主要运行时配置：

- `APP_ENV`
- `PORT`
- `DATABASE_URL`
- `JWT_SECRET`
- `REDIS_*`
- `OPENAI_*`
- `GITHUB_*`

本地开发可使用：

- `backend/.env.local`

共享环境与生产环境建议由部署系统注入环境变量，不依赖仓库内真实 `.env` 文件。

### Frontend

frontend 只允许公开配置：

- `VITE_APP_ENV`
- `VITE_API_BASE_URL`

前端不会持有：

- 数据库连接串
- JWT secret
- OpenAI key
- GitHub token

## Startup Model

### Local Startup

1. `./scripts/db-local.sh`
2. `./scripts/run-all-migrations.sh`
3. `./scripts/seed-local.sh`（可选）
4. `cd backend && go run cmd/server/main.go`
5. `cd frontend && npm run dev`

### Deployment Startup

固定顺序：

1. 执行 migration
2. 发布 backend
3. 发布 frontend

即：

```text
migrate -> backend -> frontend
```

## Migration Model

### Principles

- 不允许依赖应用启动自动改表
- 新增 schema 变化必须是显式 migration
- 默认优先非破坏性变更
- 破坏性变更采用 expand-and-contract

### Example Change Flow

如果未来要替换某个字段：

1. 新增新列
2. 回填旧数据
3. backend 切换到新列
4. frontend 如有需要再跟进
5. 稳定后再删旧列

## Local Infrastructure

本地基础设施定义在：

- `infra/docker/compose.local.yml`

负责：

- PostgreSQL
- Redis

根目录的 `docker-compose.yml` 也保持为本地基础设施入口，不再承载“后端自带数据库”的旧模型。

## Verification Path

### Application

- backend 测试要求 PostgreSQL 可访问，但不要求 frontend/backend dev server 正在运行
- `cd backend && go test ./...`
- `cd frontend && npm run build`

### Database Workflow

- `./scripts/db-local.sh`
- `./scripts/run-all-migrations.sh`
- `./scripts/seed-local.sh`
- `./scripts/clear-db-data.sh`

## Repository References

- [README.md](./README.md)
- [DEPLOYMENT.md](./DEPLOYMENT.md)
- [2026-03-08-postgresql-migration-architecture-design.md](./docs/plans/2026-03-08-postgresql-migration-architecture-design.md)
- [2026-03-08-postgresql-migration-implementation-plan.md](./docs/plans/2026-03-08-postgresql-migration-implementation-plan.md)
