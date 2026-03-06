# Skill Hub — AI 驱动的技能共享平台

> React + Go 全栈项目 | 支持 Skill/MCP/Rules/Tools 上传、展示、下载 | AI 审核 & 推荐 | GitHub 自动同步

## 快速开始

### 方式一：Docker 一键启动（推荐）

```bash
# 1. 配置环境变量
cp backend/.env.example backend/.env
# 编辑 backend/.env，填写 OPENAI_API_KEY、GitHub 等配置

# 2. 一键启动前后端
docker compose up -d

# 访问 http://localhost:3000
```

单独启动：

```bash
# 只启动后端
docker compose up -d backend

# 只启动前端
docker compose up -d frontend
```

管理命令：

```bash
# 查看日志
docker compose logs -f

# 停止服务
docker compose down

# 重新构建并启动
docker compose up -d --build
```

### 方式二：本地开发

#### 环境要求
- **Go** 1.21+
- **Node.js** 18+
- **OpenAI API Key**（可选，不配置则 AI 功能降级）

#### 1. 启动后端

```bash
cd backend
cp .env.example .env
# 编辑 .env 配置

go run cmd/server/main.go
# 服务运行在 http://localhost:8080
```

#### 2. 启动前端

```bash
cd frontend
npm install
npm run dev
# 打开 http://localhost:5173
```

## 核心功能

| 功能 | 说明 |
|------|------|
| 📤 资源上传 | 支持单文件和文件夹上传，自定义缩略图（可选） |
| 🔄 GitHub 同步 | Skill 类型上传后自动推送到 GitHub 仓库，删除时同步删除 |
| 🤖 AI 审核 | 上传时自动调用 OpenAI 审核内容质量 |
| 💬 AI 对话推荐 | 首页常驻 AI 助手，SSE 流式回复 |
| 🔥 趋势榜单 | 下载量 Top 10 实时排行 |
| 📥 下载计数 | 下载自动计数统计 |
| 🗑️ 删除管理 | 删除 Skill 同步清理 GitHub 仓库文件 |
| 🖼️ 自动缩略图 | 无上传缩略图时纯代码生成（渐变+首字母） |

## 资源类型

| 类型 | 说明 | GitHub 同步 |
|------|------|:-----------:|
| Skill | 自动化技能脚本 | ✅ |
| MCP | Model Context Protocol 服务 | ❌ |
| Rules | 规则与约束配置 | ❌ |
| Tools | 开发与运维工具 | ❌ |

## 技术栈

| 层级 | 技术 | 许可证 |
|------|------|--------|
| 前端 | React 18 + Vite + TypeScript | MIT |
| 路由 | React Router v6 | MIT |
| 后端 | Go + Gin | MIT |
| ORM | GORM + SQLite | MIT |
| 部署 | Docker + Nginx | - |

## API 端点

```
GET    /api/skills              # 获取列表（?search=&category=&resource_type=&page=&page_size=）
GET    /api/skills/:id          # 获取详情
POST   /api/skills              # 上传（multipart/form-data，支持 upload_mode=file|folder）
DELETE /api/skills/:id          # 删除（同步删除 GitHub）
GET    /api/skills/:id/download # 下载文件
GET    /api/skills/trending     # 趋势榜单（?limit=10&resource_type=）
GET    /api/categories          # 获取分类列表（?resource_type=）
GET    /api/thumbnails/:file    # 获取缩略图
POST   /api/ai/chat             # AI 对话推荐（SSE 流式）
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | 8080 | 后端端口 |
| `OPENAI_API_KEY` | — | OpenAI API Key |
| `OPENAI_BASE_URL` | https://api.openai.com/v1 | API 端点 |
| `OPENAI_MODEL` | gpt-4o-mini | AI 调用模型 |
| `GITHUB_SYNC_ENABLED` | false | 是否启用 Skill 上传后自动推送 GitHub |
| `GITHUB_TOKEN` | — | GitHub PAT（需 Contents read/write 权限） |
| `GITHUB_OWNER` | — | GitHub 用户/组织名 |
| `GITHUB_REPO` | — | 目标仓库名 |
| `GITHUB_BRANCH` | main | 推送分支 |
| `GITHUB_BASE_DIR` | skills | 仓库内基础目录 |
| `DB_PATH` | ./skill_hub.db | SQLite 数据库路径 |
| `UPLOAD_DIR` | ./uploads | 文件上传路径 |
| `THUMBNAIL_DIR` | ./thumbnails | 缩略图存储路径 |

## GitHub 同步

仅 **Skill** 类型资源会同步到 GitHub，路径结构：

```
<GITHUB_BASE_DIR>/<skill-name>/file.md          # 单文件
<GITHUB_BASE_DIR>/<skill-name>/README.md         # 文件夹上传
<GITHUB_BASE_DIR>/<skill-name>/src/main.md       # 文件夹内子目录
```

详细配置说明见 [GITHUB_SYNC_SETUP.md](./GITHUB_SYNC_SETUP.md)。

## 项目结构

```
Skill_Hub/
├── docker-compose.yml          # Docker 编排
├── backend/
│   ├── Dockerfile
│   ├── .env.example
│   ├── cmd/server/main.go      # 入口
│   └── internal/
│       ├── config/             # 配置加载
│       ├── handler/            # HTTP 处理器
│       ├── middleware/         # CORS 中间件
│       ├── model/              # 数据模型
│       └── service/            # 业务逻辑 & GitHub 同步
└── frontend/
    ├── Dockerfile
    ├── nginx.conf              # Nginx 配置
    └── src/
        ├── pages/              # 页面组件
        ├── components/         # UI 组件
        └── services/           # API 调用
```
