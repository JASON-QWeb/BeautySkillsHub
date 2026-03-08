# Skill Hub Deployment

本文档是一份可直接照着操作的部署手册，目标是把当前仓库部署到共享环境或生产环境。

核心原则只有一条：

```text
migrate -> backend -> frontend
```

不要让 backend 在启动时自动改表。

## 1. 推荐部署形态

推荐形态：

- 外部 PostgreSQL
- 可选外部 Redis
- backend 单独部署
- frontend 单独部署
- 发布前先执行 migration

如果你当前代码托管在 GitHub，可以先依赖：

- [`.github/workflows/verify.yml`](./.github/workflows/verify.yml)

完成基础校验，再进入正式发布。

## 2. 部署前准备

### 系统要求

建议环境：

- Linux x86_64
- Docker Engine（如果采用容器化部署）
- Go `1.25+`（如果在服务器上直接跑 migration 或 backend）
- Node.js `20+`（如果在服务器上直接构建 frontend）
- 可访问 PostgreSQL
- 若启用 OpenAI / GitHub 集成，需要能访问对应外部 API

### 基础资源

部署前至少准备：

- PostgreSQL 数据库实例
- 一个可连接 PostgreSQL 的 `DATABASE_URL`
- backend 运行时环境变量
- frontend 的公开配置

可选：

- Redis
- 外层网关 / Nginx / TLS
- 监控与日志采集

## 3. backend 环境变量

### 必需

- `APP_ENV`
- `PORT`
- `DATABASE_URL`
- `JWT_SECRET`

### 按需

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

### 推荐做法

生产不要依赖仓库里的 `.env.local`。建议把 backend 环境变量放在：

- 部署系统注入
- Secret 管理平台
- 或服务器上的独立 env 文件，例如 `/etc/skill-hub/backend.env`

示例：

```env
APP_ENV=prod
PORT=8080
DATABASE_URL=postgres://skillhub:strong-password@postgres-host:5432/skillhub_prod?sslmode=disable
JWT_SECRET=replace-with-a-strong-secret
REDIS_ADDR=redis-host:6379
REDIS_PASSWORD=
REDIS_DB=0
OPENAI_API_KEY=
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4o-mini
GITHUB_SYNC_ENABLED=false
```

## 4. frontend 配置

frontend 只需要公开配置：

- `VITE_APP_ENV`
- `VITE_API_BASE_URL`

当前仓库有两种典型部署方式：

### 方式 A：容器内 Nginx 反代 backend

如果直接使用 [frontend/Dockerfile](/Users/qianjianghao/Desktop/Skill_Hub/frontend/Dockerfile) 和 [frontend/nginx.conf](/Users/qianjianghao/Desktop/Skill_Hub/frontend/nginx.conf)，那么前端容器会把 `/api/` 代理到容器网络里的 `backend:8080`。

这时前端不一定需要显式设置 `VITE_API_BASE_URL`。

### 方式 B：静态资源交给外部 Nginx / CDN

如果你把 `frontend/dist` 部署到独立静态服务器，而不是使用仓库内的前端容器，那么建议在构建前设置：

```bash
export VITE_APP_ENV=prod
export VITE_API_BASE_URL=https://your-backend.example.com
```

然后再执行：

```bash
cd frontend && npm ci && npm run build
```

## 5. 执行 migration

这是部署里最关键的一步。

在目标环境准备好 `DATABASE_URL` 后，先执行：

```bash
export DATABASE_URL='postgres://skillhub:strong-password@postgres-host:5432/skillhub_prod?sslmode=disable'
./scripts/run-all-migrations.sh
```

要求：

- migration 失败，后续发布立即停止
- migration 成功后，才允许发布 backend
- backend 不允许代替 migration 做 schema 变更

## 6. 部署 backend

## 6.1 方式 A：容器化部署 backend

仓库已经提供：

- [backend/Dockerfile](/Users/qianjianghao/Desktop/Skill_Hub/backend/Dockerfile)

推荐命令：

```bash
docker network create skill-hub || true

docker volume create skillhub_uploads
docker volume create skillhub_thumbnails
docker volume create skillhub_avatars

docker build -t skill-hub-backend:latest ./backend

docker rm -f backend || true
docker run -d \
  --name backend \
  --restart unless-stopped \
  --network skill-hub \
  -p 8080:8080 \
  --env-file /etc/skill-hub/backend.env \
  -v skillhub_uploads:/app/uploads \
  -v skillhub_thumbnails:/app/thumbnails \
  -v skillhub_avatars:/app/avatars \
  skill-hub-backend:latest
```

说明：

- backend 容器名建议就叫 `backend`
- 这样前端容器可以直接通过容器网络里的 `backend:8080` 反代 API
- 上传目录、缩略图、头像目录要用 volume 持久化

## 6.2 方式 B：直接部署 backend 二进制

如果你不使用 Docker，也可以：

```bash
cd backend
go build -o ../bin/skill-hub-backend ./cmd/server
```

然后通过 systemd 或其他进程管理工具启动，并注入 backend 环境变量。

## 7. 部署 frontend

## 7.1 方式 A：容器化部署 frontend

仓库已经提供：

- [frontend/Dockerfile](/Users/qianjianghao/Desktop/Skill_Hub/frontend/Dockerfile)
- [frontend/nginx.conf](/Users/qianjianghao/Desktop/Skill_Hub/frontend/nginx.conf)

推荐命令：

```bash
docker build -t skill-hub-frontend:latest ./frontend

docker rm -f frontend || true
docker run -d \
  --name frontend \
  --restart unless-stopped \
  --network skill-hub \
  -p 80:80 \
  skill-hub-frontend:latest
```

说明：

- 这个前端容器默认会把 `/api/` 转发给同一 Docker 网络里的 `backend:8080`
- 因此 backend 容器名和网络连通性很关键

## 7.2 方式 B：部署静态产物

如果你不用前端容器，则按下面方式：

```bash
export VITE_APP_ENV=prod
export VITE_API_BASE_URL=https://your-backend.example.com
cd frontend && npm ci && npm run build
```

然后把 `frontend/dist` 交给你自己的 Nginx、对象存储或 CDN。

## 8. 完整部署顺序

推荐顺序：

1. 拉取或更新代码
2. 准备 backend 环境变量
3. 确认 PostgreSQL 可连接
4. 执行 migration
5. 发布 backend
6. 验证 backend
7. 发布 frontend
8. 验证 frontend 与 API

也就是：

```text
update code -> migrate -> backend -> frontend -> verify
```

## 9. 发布后验证

最低验证清单：

```bash
curl -i http://127.0.0.1/api/skills
curl -I http://127.0.0.1/
docker logs backend --tail 100
```

期望：

- frontend 首页可以访问
- `/api/skills` 返回 `200`
- backend 日志没有数据库连接或 migration 相关错误

## 10. 升级发布

以后每次升级建议按下面流程：

```bash
git pull
export DATABASE_URL='postgres://skillhub:strong-password@postgres-host:5432/skillhub_prod?sslmode=disable'
./scripts/run-all-migrations.sh

docker build -t skill-hub-backend:latest ./backend
docker build -t skill-hub-frontend:latest ./frontend

docker rm -f backend frontend || true
# 然后重新 run backend / frontend
```

如果你采用 systemd 或平台型部署，也同样遵守：

```text
migrate -> backend -> frontend
```

## 11. 回滚原则

推荐把回滚分成两层：

### 应用回滚

- 回滚 backend 版本
- 回滚 frontend 版本

### 数据库回滚

- 不建议把 `down` migration 当常规回滚按钮
- 生产环境更推荐：
  - 前向修复
  - 兼容性发布
  - 稳定后再清理旧结构

## 12. 常见问题

### 12.1 migration 失败

处理顺序：

1. 停止后续发布
2. 查看失败的 migration 文件
3. 修复后重新执行 migration
4. 确认 schema 状态正确后再继续发布

### 12.2 backend 启动失败

检查：

1. `DATABASE_URL` 是否正确
2. migration 是否已成功执行
3. PostgreSQL 网络是否可达
4. backend env 文件是否加载成功

### 12.3 frontend 调不到接口

检查：

1. `VITE_API_BASE_URL` 是否正确
2. 如果是前端容器，backend 是否在同一 Docker 网络里并且名字为 `backend`
3. 反向代理、网关或安全组是否已放通

### 12.4 GitHub Actions 和正式部署的关系

仓库内的 GitHub Actions workflow 只负责：

- backend test
- frontend build
- 基础 verify

它不是完整部署流水线。正式部署仍然要遵守本文件里的发布顺序。

## 13. 相关文档

- [README.md](./README.md)
- [ARCHITECTURE.md](./ARCHITECTURE.md)
- [GITHUB_ACTIONS.md](./GITHUB_ACTIONS.md)
- [CI_CD_TEMPLATE.md](./CI_CD_TEMPLATE.md)
