# Skill Hub Deployment

本文档描述当前仓库部署到共享环境或生产环境时的推荐做法。

核心原则：

```text
migrate -> backend -> frontend
```

不要让 backend 在启动时替代 migration 做 schema 变更。

## 1. 推荐部署形态

推荐使用：

- 外部 PostgreSQL
- 可选外部 Redis
- backend 单独部署
- frontend 单独部署
- 发布前先执行 migration

如果你使用容器化部署，仓库已经提供：

- [backend/Dockerfile](./backend/Dockerfile)
- [frontend/Dockerfile](./frontend/Dockerfile)
- [frontend/nginx.conf](./frontend/nginx.conf)

## 2. 部署前准备

### 必备资源

- PostgreSQL 数据库实例
- 安全的 `DATABASE_URL`
- backend 运行时环境变量
- frontend 构建配置

### 可选资源

- Redis
- 反向代理 / TLS 终止层
- 日志平台 / 监控平台

## 3. 生产安全约束

### 3.1 DATABASE_URL

非本地环境当前会强制要求：

- `DATABASE_URL` 不能为空
- `DATABASE_URL` 不能使用 `sslmode=disable`

所以生产示例应使用：

- `sslmode=require`
- 或更严格的 `sslmode=verify-full`

示例：

```env
DATABASE_URL=postgres://skillhub:strong-password@postgres-host:5432/skillhub_prod?sslmode=require
```

### 3.2 JWT_SECRET

非本地环境必须显式设置：

```env
JWT_SECRET=replace-with-a-long-random-secret
```

未设置时服务会直接启动失败，而不是生成临时密钥。

### 3.3 CORS / 安全头

生产至少应配置：

- `CORS_ALLOWED_ORIGINS`
- `SECURITY_CSP`
- `SECURITY_CSP_REPORT_ONLY`（灰度期间可先开）
- `HSTS_ENABLED=true`

### 3.4 限流

当前后端已经内建以下速率限制：

- 登录
- 注册
- 审核重试
- AI chat

生产建议同时配置 Redis，让限流在多实例下可共享状态。

## 4. backend 环境变量

### 必需

- `APP_ENV`
- `PORT`
- `DATABASE_URL`
- `JWT_SECRET`

### 常用可选

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
- `CORS_ALLOWED_ORIGINS`
- `SECURITY_CSP`
- `SECURITY_CSP_REPORT_ONLY`
- `HSTS_ENABLED`

推荐示例：

```env
APP_ENV=production
PORT=8080
DATABASE_URL=postgres://skillhub:strong-password@postgres-host:5432/skillhub_prod?sslmode=require
JWT_SECRET=replace-with-a-long-random-secret
REDIS_ADDR=redis-host:6379
REDIS_PASSWORD=
REDIS_DB=0
OPENAI_API_KEY=
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4o-mini
GITHUB_SYNC_ENABLED=false
CORS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
SECURITY_CSP=default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'
SECURITY_CSP_REPORT_ONLY=false
HSTS_ENABLED=true
```

## 5. frontend 配置

frontend 只接受公开配置：

- `VITE_APP_ENV`
- `VITE_API_BASE_URL`

两种典型部署方式：

### 5.1 前端容器内 Nginx 反代 backend

如果直接使用仓库里的前端容器：

- 容器内部 Nginx 监听 `8080`
- `/api/` 会转发给同一 Docker 网络内的 `backend:8080`

### 5.2 纯静态部署

如果把 `frontend/dist` 交给外部 Nginx / CDN：

```bash
export VITE_APP_ENV=production
export VITE_API_BASE_URL=https://your-backend.example.com
cd frontend && npm ci && npm run build
```

## 6. 执行 migration

部署前先执行：

```bash
export DATABASE_URL='postgres://skillhub:strong-password@postgres-host:5432/skillhub_prod?sslmode=require'
./scripts/run-all-migrations.sh
```

要求：

- migration 失败，后续发布立即停止
- migration 成功后，才允许发布 backend

## 7. 部署 backend

### 7.1 容器化部署

当前 backend 镜像特性：

- 进程以非 root 用户 `skillhub` 运行
- 端口 `8080`
- `/health` 可作为健康检查

示例：

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

### 7.2 非容器部署

```bash
cd backend
go build -o ../bin/skill-hub-backend ./cmd/server
```

然后通过 systemd 或平台进程管理工具启动，并注入运行时环境变量。

## 8. 部署 frontend

### 8.1 容器化部署

当前 frontend 镜像特性：

- 容器以非 root 用户 `nginx` 运行
- 容器内监听 `8080`
- 外部通常映射到 `80`

示例：

```bash
docker build -t skill-hub-frontend:latest ./frontend

docker rm -f frontend || true
docker run -d \
  --name frontend \
  --restart unless-stopped \
  --network skill-hub \
  -p 80:8080 \
  skill-hub-frontend:latest
```

### 8.2 静态产物部署

```bash
export VITE_APP_ENV=production
export VITE_API_BASE_URL=https://your-backend.example.com
cd frontend && npm ci && npm run build
```

然后把 `frontend/dist` 交给你自己的静态托管层。

## 9. 健康检查与发布后验证

最低验证清单：

```bash
curl -i http://127.0.0.1:8080/health
curl -i http://127.0.0.1:8080/api/skills
curl -I http://127.0.0.1/
docker logs backend --tail 100
```

期望：

- `/health` 返回 `200`
- `/api/skills` 返回 `200`
- frontend 首页可访问
- backend 日志没有数据库连接、migration、JWT 或 CORS 配置错误

## 10. 推荐发布顺序

```text
update code -> migrate -> backend -> frontend -> verify
```

对应操作：

1. 更新代码
2. 准备 backend 环境变量
3. 验证 PostgreSQL 可连接
4. 执行 migration
5. 发布 backend
6. 验证 backend `/health`
7. 发布 frontend
8. 做端到端验证

## 11. 升级发布

```bash
git pull
export DATABASE_URL='postgres://skillhub:strong-password@postgres-host:5432/skillhub_prod?sslmode=require'
./scripts/run-all-migrations.sh

docker build -t skill-hub-backend:latest ./backend
docker build -t skill-hub-frontend:latest ./frontend

docker rm -f backend frontend || true
# 然后按你的运行方式重新启动
```

## 12. 回滚原则

应用回滚和数据库回滚分开看。

### 应用层

- backend 可独立回滚
- frontend 可独立回滚

### 数据库层

- 不建议把 `down` migration 当作生产常规回滚按钮
- 更推荐：
  - 兼容性发布
  - 前向修复
  - 稳定后再清理旧结构

## 13. 常见问题

### 13.1 服务启动即 panic

优先检查：

- `APP_ENV` 是否已经切到非本地环境
- `DATABASE_URL` 是否仍然使用 `sslmode=disable`
- `JWT_SECRET` 是否遗漏

### 13.2 前端跨域失败

优先检查：

- `CORS_ALLOWED_ORIGINS` 是否包含真实前端域名
- 反向代理是否改写了 `Origin`

### 13.3 审核/AI 能力不可用

优先检查：

- `OPENAI_API_KEY`
- `OPENAI_BASE_URL`
- 外网访问策略
