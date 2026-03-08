# CI/CD Template Notes

本文档不是企业内部流水线的最终配置文件，而是一份可直接映射到你们 CI/CD 平台的步骤模板。

如果你当前代码托管在 GitHub，本仓库已经有基础校验 workflow：

- `.github/workflows/verify.yml`

它覆盖的是 `verify`，不是完整 `deploy`。

## Goal

保证每次发布都遵守：

```text
verify -> migrate -> deploy backend -> deploy frontend
```

## Recommended Pipeline Stages

### 1. Verify

代码合入前或构建前执行：

```bash
./scripts/db-local.sh
./scripts/run-all-migrations.sh
cd backend && go test ./...
cd frontend && npm ci && npm run build
```

其中：

- backend 测试只依赖 PostgreSQL，不依赖 frontend/backend 进程已启动
- 如果 CI 平台已有独立 PostgreSQL 服务，可直接注入 `DATABASE_URL`，不必使用本地脚本

如果你们平台支持，也可以增加：

```bash
bash -n scripts/db-local.sh scripts/run-all-migrations.sh scripts/seed-local.sh scripts/clear-db-data.sh
docker compose -f infra/docker/compose.local.yml config
```

## 2. Package

按你们企业现有方式构建：

- backend 镜像或二进制
- frontend 静态产物或镜像

这里不需要让 frontend 感知任何 secret。

## 3. Migrate

在目标环境注入：

- `DATABASE_URL`

然后执行：

```bash
./scripts/run-all-migrations.sh
```

要求：

- migration 失败，流水线必须立即终止
- 不允许跳过 migration 直接发布 backend

## 4. Deploy Backend

backend 发布时注入：

- `APP_ENV`
- `PORT`
- `DATABASE_URL`
- `JWT_SECRET`
- `REDIS_*`
- `OPENAI_*`
- `GITHUB_*`

backend 启动后只做运行，不做 schema 自动变更。

## 5. Deploy Frontend

frontend 发布时只注入公开变量：

- `VITE_APP_ENV`
- `VITE_API_BASE_URL`

## Environment Mapping

建议至少区分：

- `local`
- `dev`
- `stg`
- `prod`

其中：

- `local` 使用项目脚本与本地 Docker
- `dev/stg/prod` 使用独立 PostgreSQL

## Release Safety Rules

### Required

1. migration 先于 backend
2. backend 先于 frontend
3. 迁移默认采用非破坏性模式
4. 任何破坏性 migration 必须拆阶段发布

### Strongly Recommended

1. 在 `stg` 先跑一遍 migration
2. 对 PostgreSQL 做备份或快照
3. 每次发布保留 migration 执行日志

## Suggested Secret Sources

根据你们企业平台选择一种：

- 平台 Secret 管理
- CI/CD Secret
- Vault
- Kubernetes Secret / External Secret

不要把真实值提交到仓库。

## Minimal Example Flow

```text
PR -> backend test + frontend build
merge -> build artifacts
deploy dev -> run migration -> deploy backend -> deploy frontend
deploy stg -> run migration -> deploy backend -> deploy frontend
deploy prod -> backup -> run migration -> deploy backend -> deploy frontend
```

## Rollback Notes

推荐理解为两层 rollback：

1. `应用回滚`
   - 回滚 backend/frontend 版本
2. `数据库回滚`
   - 只在确认 migration 可安全回退时执行 `down`

很多企业生产环境不会轻易执行 schema `down`，而是优先：

- 前向修复
- 兼容性发布
- 再做后续清理

## Repository References

- [README.md](./README.md)
- [DEPLOYMENT.md](./DEPLOYMENT.md)
- [ARCHITECTURE.md](./ARCHITECTURE.md)
