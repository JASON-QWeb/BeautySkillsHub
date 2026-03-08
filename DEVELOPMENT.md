# Skill Hub Development

本文档面向日常维护者，说明如何在本地启动、测试、迁移和清理当前项目。

## 1. 前置要求

建议本地准备：

- Docker Desktop / Docker Engine
- Go `1.25+`
- Node.js `20+`
- npm `10+`
- `psql` 可选

## 2. 环境文件

如果你要跑宿主机开发流，先准备：

```bash
cp backend/.env.local.example backend/.env.local
cp frontend/.env.local.example frontend/.env.local
```

关键文件：

- [backend/.env.local.example](./backend/.env.local.example)
- [frontend/.env.local.example](./frontend/.env.local.example)

宿主机开发默认值里：

- `DATABASE_URL` 指向本地 PostgreSQL
- `JWT_SECRET` 使用本地开发值
- `VITE_API_BASE_URL` 指向 `http://localhost:8080`

## 3. 本地开发方式

### 3.1 方式 A：Docker Compose 完整本地栈

最省事的方式：

```bash
docker compose up -d --build
```

这一条命令默认就能启动完整本地栈：

- 不依赖 `backend/.env.local`
- backend 会拿 compose 内置的本地安全默认值
- 需要接自己的 OpenAI / GitHub 凭证时，再通过 shell 环境变量覆盖

这会启动：

- `postgres`
- `redis`
- `migrate`
- `backend`
- `frontend`

查看日志：

```bash
docker compose logs -f migrate backend frontend
```

停止：

```bash
docker compose down
```

清空 compose 数据卷：

```bash
docker compose down -v
```

### 3.2 方式 B：数据库走 Docker，前后端走宿主机进程

这是最适合本地调试代码的方式。

#### Step 1：启动 PostgreSQL 和 Redis

```bash
./scripts/db-local.sh
```

底层编排文件：

- [infra/docker/compose.local.yml](./infra/docker/compose.local.yml)

#### Step 2：执行 migration

```bash
./scripts/run-all-migrations.sh
```

#### Step 3：可选灌本地 seed

```bash
./scripts/seed-local.sh
```

#### Step 4：启动 backend

```bash
cd backend
go run ./cmd/server/main.go
```

#### Step 5：启动 frontend

```bash
cd frontend
npm ci
npm run dev
```

### 3.3 方式 C：一个终端启动宿主机开发流

```bash
./scripts/dev-all.sh
```

默认行为：

- 启动 PostgreSQL / Redis
- 执行 migration
- 执行本地 seed
- 启动 backend
- 启动 frontend

如果不想自动 seed：

```bash
SEED_LOCAL=0 ./scripts/dev-all.sh
```

## 4. 本地访问地址

- frontend：`http://localhost:5173`
- backend API：`http://localhost:8080/api/...`
- backend health：`http://localhost:8080/health`
- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`

## 5. 常用脚本

### 启动本地数据库

```bash
./scripts/db-local.sh
```

### 执行全部 migration

```bash
./scripts/run-all-migrations.sh
```

### 应用本地 seed

```bash
./scripts/seed-local.sh
```

### 清空本地业务数据

```bash
./scripts/clear-db-data.sh
```

这个脚本会：

- 调用 [backend/cmd/clear-db](./backend/cmd/clear-db)
- 清空业务表数据
- 清空 `backend/uploads`、`backend/thumbnails`、`backend/avatars`

## 6. 测试与验证

### backend 全量测试

```bash
./scripts/db-local.sh
./scripts/run-all-migrations.sh
cd backend && go test ./...
```

说明：

- backend 测试统一跑 PostgreSQL
- 使用真实 migration 建 schema
- 不需要先启动 `go run ./cmd/server/main.go`

### frontend 构建校验

```bash
cd frontend && npm ci && npm run build
```

### frontend 轻量 node 测试

```bash
node --test frontend/src/services/api/request.test.ts \
  frontend/src/features/skill-detail/readmeCache.test.ts \
  frontend/src/contexts/dialogKeydown.test.ts \
  frontend/src/components/aiMouseTracking.test.ts \
  frontend/src/features/profile/profileActivity.test.ts \
  frontend/src/features/upload/shared/tagInput.test.ts \
  frontend/docker-runtime.test.mjs
```

### Docker 镜像验证

```bash
docker build -f backend/Dockerfile backend
docker build -f frontend/Dockerfile frontend
```

## 7. 日常开发约束

- 新数据库变更只通过 `db/migrations/*.sql`
- 不要修改已上线 migration 的历史内容
- `skill / rules` 更新会进入 revision + review 流
- `mcp / tools` 更新是直接更新当前资源，不创建待审核 revision
- `docker compose up -d --build` 不依赖 `backend/.env.local`
- 宿主机直接跑 backend 时可以使用 `backend/.env.local`
- 生产不要依赖仓库内真实 env 文件

## 8. 开发时常见入口

- API 启动入口：[backend/cmd/server/main.go](./backend/cmd/server/main.go)
- migration 入口：[backend/cmd/migrate/main.go](./backend/cmd/migrate/main.go)
- profile 上传分页接口：[backend/internal/handler/profile_handlers.go](./backend/internal/handler/profile_handlers.go)
- 前端请求层：[frontend/src/services/api/request.ts](./frontend/src/services/api/request.ts)
- AI 审核流程：[ai-review流程.md](./ai-review流程.md)

## 9. 故障排查

### migration 报错

优先检查：

- `DATABASE_URL` 是否正确
- PostgreSQL 是否已启动
- 目标库里是否已有手工改过的 schema

### backend 启动 panic

如果 `APP_ENV` 不是 `local`，现在会强制校验：

- `DATABASE_URL` 不能为空
- `DATABASE_URL` 不能使用 `sslmode=disable`
- `JWT_SECRET` 必须显式设置

### frontend 登录态异常

当前前端会：

- 启动时解析 JWT `exp`
- 鉴权请求遇到 `401` 自动登出

所以如果你手工改了 token 或后端 secret，本地页面可能会直接退出登录，这是预期行为。
