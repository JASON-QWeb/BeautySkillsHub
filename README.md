<div align="center">

# BeautySkillsHub

### AI 相关资源整合平台模板

[English](./README.en.md) | 简体中文

</div>

<p align="center">
  <img src="./demo1.png" alt="BeautySkillsHub demo screenshot 1" width="49%" />
  <img src="./demo2.png" alt="BeautySkillsHub demo screenshot 2" width="49%" />
</p>

<p align="center">
  AI 相关资源整合平台模板，内置 Skill 上传 GitHub、AI Review、AI 助手等能力，开箱即用。
  <br />
  帮你快速构建聚合 <code>skill</code>、<code>rules</code>、<code>mcp</code>、<code>tools</code> 等多类资源的 AI 共享平台，B 端与 C 端场景都可直接扩展。
</p>

<p align="center">
  <a href="./AIREAD.md"><strong>AI 快速读项目</strong></a>
  ·
  <a href="#项目一键启动"><strong>快速启动</strong></a>
  ·
  <a href="#项目介绍"><strong>项目介绍</strong></a>
  ·
  <a href="#仓库导览"><strong>仓库导览</strong></a>
</p>

## 推荐：AI 一键了解项目并启动

AI 可直接阅读 [AIREAD.md](./AIREAD.md)，快速理解项目结构、运行方式与开发入口。

## 项目一键启动

推荐直接启动完整本地栈：

```bash
docker compose up -d --build
```

启动后默认访问：

- 前端：`http://localhost:5173`
- 后端健康检查：`http://localhost:8080/health`
- 后端 API：`http://localhost:8080/api/...`

停止服务：

```bash
docker compose down
```

如果你想在宿主机直接调试前后端：

```bash
./scripts/local.sh dev
```

## 项目介绍

BeautySkillsHub 是一套前后端分离的资源平台，围绕“上传、审核、发布、发现、复用”这一完整链路设计，当前已具备这些核心能力：

- `skill / rules` 走 AI 审核 + 人工复核 + revision 更新流
- `mcp / tools` 走自动通过、自动发布流
- 支持点赞、收藏、下载统计和个人上传分页
- 使用 PostgreSQL migration-first 模式管理业务 schema
- 内置 `/health`、安全响应头、CORS allowlist、限流和非 root 容器运行

技术栈：

- 后端：Go + Gin + GORM
- 前端：React + Vite + TypeScript
- 数据层：PostgreSQL + Redis
- 运行方式：Docker Compose / 宿主机双模式

## 仓库导览

- [AIREAD.md](./AIREAD.md)：给 AI 编码助手的最短上手路径
- [backend/README.md](./backend/README.md)：后端结构、启动、测试
- [frontend/README.md](./frontend/README.md)：前端结构、构建、测试
- [scripts/README.md](./scripts/README.md)：本地脚本入口说明
- [db/SCHEMA.md](./db/SCHEMA.md)：数据库结构与 migration 说明
