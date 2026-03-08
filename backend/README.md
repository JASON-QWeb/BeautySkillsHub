# Backend

`/backend` 是 Skill Hub 的 Go 后端，负责：

- 资源上传、读取、删除、下载统计
- `skill / rules` 的 AI 审核、人工复核和 revision 流程
- `mcp / tools` 的自动发布流
- 用户认证、点赞、收藏、个人上传分页
- 安全中间件、限流、健康检查

## 目录结构

- [cmd/server](./cmd/server)
  - HTTP 服务入口
- [cmd/migrate](./cmd/migrate)
  - SQL migration 执行入口
- [cmd/clear-db](./cmd/clear-db)
  - 清空业务数据的维护命令
- [internal/config](./internal/config)
  - 环境变量和运行时安全校验
- [internal/handler](./internal/handler)
  - HTTP handler 和请求校验
- [internal/service](./internal/service)
  - 业务逻辑、GitHub 同步、AI、缩略图等
- [internal/middleware](./internal/middleware)
  - CORS、安全头、限流、上传体积限制
- [internal/model](./internal/model)
  - GORM 模型

## 本地运行

### 启动服务

```bash
cd backend
go run ./cmd/server/main.go
```

如果你需要先把本地 PostgreSQL/Redis 拉起来，优先从仓库根目录执行：

```bash
./scripts/local.sh db up
./scripts/local.sh migrate
```

### 执行 migration

```bash
cd backend
go run ./cmd/migrate -database-url "$DATABASE_URL" -migrations-dir ../db/migrations
```

或在仓库根目录执行统一入口：

```bash
./scripts/local.sh migrate
```

### 清空业务数据

```bash
cd backend
go run ./cmd/clear-db
```

或在仓库根目录执行：

```bash
./scripts/local.sh reset
```

## 测试

```bash
cd backend
go test ./...
```

## 关键环境变量

- `APP_ENV`
- `PORT`
- `DATABASE_URL`
- `JWT_SECRET`
- `OPENAI_API_KEY`
- `REDIS_ADDR`
- `GITHUB_SYNC_ENABLED`

说明：

- `local` 环境下允许回退到本地默认 `DATABASE_URL`
- 非 `local` 环境必须显式提供安全 `DATABASE_URL`
- 非 `local` 环境必须显式提供 `JWT_SECRET`

## 运行时输出目录

- [uploads](./uploads)
- [thumbnails](./thumbnails)
- [avatars](./avatars)

这些目录会在本地开发和 Docker 运行时被写入，不属于业务源码的一部分。
