# Skill Hub 部署指南

本文档描述 PostgreSQL + migration-first 的部署流程。

## 1. 目标部署模型

共享环境和生产环境统一采用：

- 外部 PostgreSQL
- 可选 Redis
- backend 运行时环境变量注入
- frontend 公开环境变量注入
- 先执行 migration，再发布应用

固定顺序：

```text
migrate -> backend -> frontend
```

## 2. 服务器准备

建议环境：

- Linux x86_64
- Go 1.25+（如果在服务器上直接跑 migration 脚本）
- Node.js 20+（如果在服务器上直接构建 frontend）
- 可访问 PostgreSQL
- 若启用 OpenAI/GitHub 集成，则需访问对应外部 API

如果采用容器化部署，请确保：

- backend 容器能访问 PostgreSQL
- frontend 只暴露公开配置
- migration 作为独立步骤执行，而不是由 backend 启动触发

## 3. 环境变量

### Backend 必需

- `APP_ENV`
- `PORT`
- `DATABASE_URL`
- `JWT_SECRET`

### Backend 按需

- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `OPENAI_API_KEY`
- `OPENAI_BASE_URL`
- `OPENAI_MODEL`
- `GITHUB_SYNC_ENABLED`
- `GITHUB_TOKEN`
- `GITHUB_OWNER`
- `GITHUB_REPO`
- `GITHUB_BRANCH`
- `GITHUB_BASE_DIR`

### Frontend 公开变量

- `VITE_APP_ENV`
- `VITE_API_BASE_URL`

不要把任何 secret 放进 frontend。

## 4. 发布前准备

1. 创建或确认目标 PostgreSQL 已可用
2. 配置 `DATABASE_URL`
3. 确认 migration 文件已随代码发布
4. 确认 backend 将不再依赖启动时自动建表

如果仓库托管在 GitHub，可先让：

- `.github/workflows/verify.yml`

完成基础校验，再进入你们企业内部发布流程。

## 5. 执行 migration

在部署环境注入 `DATABASE_URL` 后执行：

```bash
./scripts/run-all-migrations.sh
```

要求：

- 失败即停止发布
- migration 成功后才能继续发布 backend

## 6. 发布 backend

确认 migration 成功后，发布 backend。

backend 启动时应只做：

- 连接数据库
- 初始化缓存/外部服务
- 提供 API

backend 不应再做：

- 自动建表
- 自动修复 schema
- 自动跑结构迁移

## 7. 发布 frontend

backend 发布成功并完成健康检查后，再发布 frontend。

frontend 只需要公开配置，例如：

- `VITE_APP_ENV=stg`
- `VITE_API_BASE_URL=https://your-backend.example.com`

## 8. 升级字段或表结构的标准流程

每次 schema 变更都按下面步骤：

1. 新建 migration 文件
2. 先在本地和测试环境验证
3. 在共享环境执行 migration
4. 发布 backend
5. 再发布 frontend（如需要）

对于破坏性变更，拆成多次发布：

1. 新增字段
2. 回填数据
3. 代码切换到新字段
4. 删除旧字段

## 9. 本地和生产的边界

本地可以用：

- `./scripts/db-local.sh`
- `./scripts/run-all-migrations.sh`
- `./scripts/seed-local.sh`

生产不要依赖：

- `backend/.env.local`
- SQLite 文件
- 本地 Docker volume 当作生产数据库

## 10. 常见问题

### 10.0 GitHub Actions 和部署是什么关系

本仓库内的 GitHub Actions workflow 只负责：

1. verify
2. backend test
3. frontend build

它不是完整部署流水线。  
正式发布仍然建议按：

```text
migrate -> backend -> frontend
```

详细说明见 [GITHUB_ACTIONS.md](./GITHUB_ACTIONS.md) 和 [CI_CD_TEMPLATE.md](./CI_CD_TEMPLATE.md)。

### 10.1 migration 执行失败

处理顺序：

1. 停止后续发布
2. 查看失败的 migration 文件
3. 修复后重新执行 migration
4. 确认 schema 状态一致后再继续发布

### 10.2 backend 启动失败

检查：

1. `DATABASE_URL` 是否正确
2. migration 是否已成功执行
3. PostgreSQL 网络是否可达

### 10.3 frontend 调不到接口

检查：

1. `VITE_API_BASE_URL` 是否正确
2. backend 是否已发布成功
3. 反向代理或网关是否已放通
