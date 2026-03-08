# GitHub Actions

本仓库当前已经提供可直接运行的 GitHub Actions 校验流水线：

- `.github/workflows/verify.yml`

## 目标

这条 workflow 只负责 `verify`，不负责 `deploy`。

它验证两件事：

1. backend 在 PostgreSQL + migration-first 模式下可以通过测试
2. frontend 可以正常安装依赖并完成生产构建

## 触发方式

当前触发条件：

- `push`
- `pull_request`
- `workflow_dispatch`

## Workflow 结构

### Backend Test

backend job 会：

1. 启动 GitHub Actions 的 PostgreSQL service container
2. 注入 `DATABASE_URL` / `TEST_DATABASE_URL`
3. 执行 `./scripts/run-all-migrations.sh`
4. 执行 `cd backend && go test ./...`

说明：

- 不启动 backend dev server
- 不依赖 frontend dev server
- 测试依赖 PostgreSQL，而不是 SQLite

### Frontend Build

frontend job 会：

1. 安装 Node.js
2. 执行 `cd frontend && npm ci`
3. 执行 `cd frontend && npm run build`

## 和企业内部 CI/CD 的关系

这个 workflow 是仓库级的基础校验流水线。

你后续在企业内网可以继续扩展成：

- `verify`
- `package`
- `migrate`
- `deploy backend`
- `deploy frontend`

企业内部更完整的落地思路，继续参考：

- [CI_CD_TEMPLATE.md](./CI_CD_TEMPLATE.md)
- [DEPLOYMENT.md](./DEPLOYMENT.md)

## 本地对应验证命令

本地想复现 GitHub Actions 的核心验证，执行：

```bash
./scripts/db-local.sh
./scripts/run-all-migrations.sh
cd backend && go test ./...
cd frontend && npm run build
```

说明：

- backend 测试不要求你先启动 `go run cmd/server/main.go`
- frontend 构建不要求你先启动 `npm run dev`
