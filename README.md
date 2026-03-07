# Skill Hub

AI 驱动的技能共享平台（React + Go），支持 Skill/MCP/Rules/Tools 上传、AI 审核、收藏点赞、下载统计、以及 Skill 到 GitHub 仓库同步。

## 快速开始

### 重制数据库

```bash
./scripts/clear-db-data.sh
```

### Docker 启动（推荐）

前提：
- Docker Desktop/Engine 已启动（`docker info` 正常）
- 端口 `3000` 可用

```bash
cp backend/.env.example backend/.env
# 编辑 backend/.env（至少建议配置 JWT_SECRET，按需配置 OPENAI / GitHub）

docker compose up -d --build
docker compose ps
docker compose logs -f backend
```

访问：
- 前端: `http://localhost:3000`
- API（经 Nginx 反代）: `http://localhost:3000/api/...`

停止：

```bash
docker compose down
```

### Development（本地开发）

环境要求：
- Go `1.25+`
- Node.js `20+`
- npm `10+`

首次准备：

```bash
cp backend/.env.example backend/.env
# 按需编辑 backend/.env（JWT_SECRET / OPENAI / GitHub）
```

启动后端（终端 1）：

```bash
cd backend
go run cmd/server/main.go
```

启动前端（终端 2）：

```bash
cd frontend
npm install
npm run dev
```

开发访问地址：
- 前端（Vite）：`http://localhost:5173`
- 后端 API：`http://localhost:8080/api/...`
- 前端已配置 `/api` 代理到 `http://localhost:8080`

常用开发命令：

```bash
# 清空本地数据（重置 SQLite + 上传目录）
./scripts/clear-db-data.sh

# 后端测试
cd backend && go test ./...

# 前端构建校验
cd frontend && npm run build
```

资源路由约定：
- 列表页：`/resource/{skill|rules|mcp|tools}`
- 上传页：`/resource/{type}/upload`
- 详情页：`/resource/{type}/{id}`
- 兼容路由：`/upload`、`/skill/:id` 会重定向到新路由

上传行为：
- `skill`：文件/文件夹上传，AI 审核 + 人工复核
- `rules`：仅 `.md/.txt` 或粘贴 Markdown，AI 审核 + 人工复核
- `mcp`：文章型发布（metadata），无 review，可填 GitHub 链接
- `tools`：文章型发布（metadata/file），无 review，可附压缩包

## 上传与 GitHub 同步规则

仅 `resource_type=skill` 会同步 GitHub（`GITHUB_SYNC_ENABLED=true` 时生效）。

路径规则：
- 单文件上传：`<GITHUB_BASE_DIR>/<技能名>/<文件名>`
- 文件夹上传：始终以“技能名”作为根目录，本地文件夹名会被忽略
- 文件夹内部层级会保留：例如 `src/main.md`

冲突策略：
- 同名技能目录已存在时，后端返回 `409`，前端提示“请修改技能名称后重试”
- 不再自动加时间戳重命名

字符策略：
- 支持中文技能名和中文文件名，不强制英文化

删除策略：
- 优先按上传时记录的 GitHub 文件清单精确删除
- 兼容老数据：无清单时回退到旧路径删除逻辑

## 核心功能

- 上传单文件/文件夹，支持自定义缩略图
- AI 审核 + AI 推荐对话
- 收藏、点赞、下载统计
- Skill 与 GitHub 双向一致性删除
- Redis 缓存可选（未配置时自动回退数据库）

## API 概览

```text
POST   /api/auth/register
POST   /api/auth/login
GET    /api/auth/me

GET    /api/skills
GET    /api/skills/:id
POST   /api/skills
PUT    /api/skills/:id
DELETE /api/skills/:id
GET    /api/skills/:id/review-status
POST   /api/skills/:id/review/retry
POST   /api/skills/:id/human-review

GET    /api/rules
GET    /api/rules/:id
POST   /api/rules
PUT    /api/rules/:id
DELETE /api/rules/:id
GET    /api/rules/:id/review-status
POST   /api/rules/:id/review/retry
POST   /api/rules/:id/human-review

GET    /api/mcps
GET    /api/mcps/:id
POST   /api/mcps
PUT    /api/mcps/:id
DELETE /api/mcps/:id

GET    /api/tools
GET    /api/tools/:id
POST   /api/tools
PUT    /api/tools/:id
DELETE /api/tools/:id

POST   /api/skills/:id/like
POST   /api/skills/:id/favorite
DELETE /api/skills/:id/favorite
GET    /api/me/favorites

GET    /api/skills/:id/download
POST   /api/skills/:id/download-hit
GET    /api/skills/trending

POST   /api/content-assets/images
GET    /api/content-assets/:filename

POST   /api/ai/chat
GET    /api/thumbnails/:filename
GET    /api/avatars/:filename
```

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | 后端端口 |
| `JWT_SECRET` | `skill-hub-default-secret-change-me` | JWT 签名密钥（生产必须修改） |
| `DB_PATH` | `./skill_hub.db` | SQLite 路径 |
| `UPLOAD_DIR` | `./uploads` | 上传目录 |
| `THUMBNAIL_DIR` | `./thumbnails` | 缩略图目录 |
| `OPENAI_API_KEY` | 空 | OpenAI Key |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | OpenAI Base URL |
| `OPENAI_MODEL` | `gpt-4o-mini` | AI 模型 |
| `GITHUB_SYNC_ENABLED` | `false` | 是否开启 Skill GitHub 同步 |
| `GITHUB_TOKEN` | 空 | GitHub PAT（contents 读写权限） |
| `GITHUB_OWNER` | 空 | 目标 owner/org |
| `GITHUB_REPO` | 空 | 目标 repo |
| `GITHUB_BRANCH` | `main` | 同步分支 |
| `GITHUB_BASE_DIR` | `skills` | 仓库根目录 |
| `REDIS_ADDR` | 空 | Redis 地址，例如 `redis:6379` |
| `REDIS_PASSWORD` | 空 | Redis 密码 |
| `REDIS_DB` | `0` | Redis DB |
| `AI_SKILLS_CACHE_KEY` | `ai:skills_context:v1` | AI 上下文缓存键 |
| `AI_SKILLS_INVALIDATE_CHANNEL` | `ai:skills_context:invalidate` | AI 上下文失效广播 |

## Docker 结构说明

- `frontend`：Nginx 提供静态页面，并将 `/api` 反代到 `backend:8080`
- `backend`：Go API + SQLite + GitHub/OpenAI 集成
- `redis`：可选缓存
- 前端字体（`Fira Sans` / `Fira Code` / `Didact Gothic`）已本地化到 `frontend/public/fonts`，运行时不依赖 Google Fonts 外链
- 数据卷：
  - `db_data` -> `/app/data`
  - `uploads` -> `/app/uploads`（含正文图片资源 `content-assets`）
  - `thumbnails` -> `/app/thumbnails`
  - `avatars` -> `/app/avatars`

## 文档

- GitHub 同步配置: [GITHUB_SYNC_SETUP.md](./GITHUB_SYNC_SETUP.md)
- 部署指南（生产环境）: [DEPLOYMENT.md](./DEPLOYMENT.md)
