# Skill Hub 部署指南

本文档给出基于 Docker Compose 的生产部署步骤。

## 1. 服务器准备

建议环境：
- Linux x86_64
- Docker Engine + Docker Compose Plugin（`docker compose version` 可用）
- 可访问外网（拉镜像；若启用 OpenAI/GitHub 集成则还需访问对应 API）

说明：
- 前端字体已随项目静态资源打包（`frontend/public/fonts`），页面渲染不依赖 Google Fonts 外链

开放端口：
- `3000/tcp`（前端入口）

## 2. 拉取代码并配置环境

```bash
git clone <your-repo-url> Skill_Hub
cd Skill_Hub
cp backend/.env.example backend/.env
```

必须修改的变量（`backend/.env`）：
- `JWT_SECRET`：生产环境必须替换为高强度随机串
- `backend/.env` 仅用于服务器本地，不要提交到 Git 仓库

按需启用：
- OpenAI：`OPENAI_API_KEY`、`OPENAI_BASE_URL`、`OPENAI_MODEL`
- GitHub 同步：`GITHUB_SYNC_ENABLED=true` + `GITHUB_TOKEN/GITHUB_OWNER/GITHUB_REPO`
- Redis：`REDIS_ADDR`（例如 `redis:6379`）

## 3. 启动服务

```bash
docker compose up -d --build
docker compose ps
docker compose logs -f backend
```

访问地址：
- `http://<server-ip>:3000`

## 4. 健康检查

### 4.1 容器状态

```bash
docker compose ps
```

预期：`frontend`、`backend`、`redis` 均为 `Up`。

### 4.2 API 检查

```bash
curl -i http://<server-ip>:3000/api/skills
```

预期：返回 `200` 与 JSON 列表结构。

## 5. 数据持久化

Compose 已配置持久化卷：
- `db_data`：SQLite 数据库
- `uploads`：上传文件
- `thumbnails`：缩略图
- `avatars`：头像

## 6. 备份与恢复

### 6.1 备份数据库

```bash
mkdir -p backups
docker run --rm \
  -v skill_hub_db_data:/data \
  -v "$(pwd)/backups:/backup" \
  alpine sh -c 'cp /data/skill_hub.db /backup/skill_hub_$(date +%F_%H%M%S).db'
```

### 6.2 恢复数据库

```bash
docker compose down
docker run --rm \
  -v skill_hub_db_data:/data \
  -v "$(pwd)/backups:/backup" \
  alpine sh -c 'cp /backup/<backup-file>.db /data/skill_hub.db'
docker compose up -d
```

## 7. 升级发布

```bash
git pull
docker compose up -d --build
docker compose logs -f backend
```

## 8. 常见问题

### 8.1 `Cannot connect to the Docker daemon`

Docker 未启动。先启动 Docker Desktop/Engine，再执行 `docker compose ...`。

### 8.2 前端能打开但接口 502/504

排查顺序：
1. `docker compose ps` 看 `backend` 是否在运行
2. `docker compose logs -f backend` 看启动报错
3. 检查 `backend/.env` 中关键变量是否缺失

### 8.3 上传 Skill 返回 409

表示 GitHub 中同名技能目录已存在。请改技能名后重试（设计行为，不自动重命名）。

### 8.4 OpenAI/GitHub 调用失败

检查：
1. Key/Token 是否正确
2. 服务器是否能访问外网
3. `backend` 日志中具体错误信息
