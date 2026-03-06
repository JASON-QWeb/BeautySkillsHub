# Skill 上传同步 GitHub 配置说明

本文说明如何把你上传的 skill 自动推送到你自己的 GitHub 仓库，以及本项目的模块化实现结构。

## 1. 功能行为

### 上传同步
- 仅 **Skill** 类型资源会同步到 GitHub，MCP/Rules/Tools 不会触发同步。
- 支持 **单文件上传** 和 **文件夹上传** 两种模式。
- 仓库路径规范：`<GITHUB_BASE_DIR>/<title-slug>/<filename>`
- 示例：
  - 单文件：`skills/my-awesome-skill/agent.md`
  - 文件夹：`skills/my-project/README.md`、`skills/my-project/src/main.md`
- 若同路径文件已存在，会使用 SHA 进行覆盖更新。
- GitHub 同步失败不会阻断上传主流程，接口仍返回 201，并在字段里标记失败原因。

### 删除同步
- 前端删除 skill 时，后端会同步删除 GitHub 仓库中对应的文件。
- 删除操作会清理该 skill 目录下的所有文件。
- GitHub 删除失败不会阻断本地删除，错误信息会在响应中返回。

## 2. GitHub 准备

1. 准备一个目标仓库（例如 `agent-skills`）。
2. 生成 PAT（Personal Access Token）。
3. 给 Token 至少以下权限：
- `Contents: Read and write`
4. 记录：
- Owner（你的用户名或组织名）
- Repo 名称
- Branch（通常是 `main`）

## 3. 后端 .env 配置

编辑 `backend/.env`：

```env
GITHUB_SYNC_ENABLED=true
GITHUB_TOKEN=ghp_xxx
GITHUB_OWNER=your-name
GITHUB_REPO=your-skill-repo
GITHUB_BRANCH=main
GITHUB_BASE_DIR=skills
```

说明：

- `GITHUB_SYNC_ENABLED=false` 时完全不调用 GitHub。
- `GITHUB_BASE_DIR` 用于统一仓库根目录，推荐保持 `skills`。

## 4. API 接口

### 上传 skill
- `POST /api/skills` - 支持 `upload_mode=file`（单文件）或 `upload_mode=folder`（文件夹）
- 文件夹模式通过 `files`（多文件）和 `file_paths`（相对路径）字段传递

### 删除 skill
- `DELETE /api/skills/:id` - 删除 skill 并同步删除 GitHub 上的文件

### 响应字段
- `github_sync_status`：`success | failed | disabled`
- `github_path`：仓库内路径
- `github_url`：GitHub 文件链接（成功时）
- `github_sync_error`：失败原因（失败时）

## 5. 启动与验证

1. 启动后端：

```bash
cd backend
go run cmd/server/main.go
```

2. 上传一个 skill（单文件或文件夹）。
3. 查看接口返回中的同步状态字段。
4. 在 GitHub 仓库中确认文件已推送到 `skills/<skill-name>/` 目录下。

## 6. 常见错误排查

- `401/403`：Token 无效或权限不足，确保 Contents 权限为 Read and write。
- `404`：`GITHUB_OWNER` / `GITHUB_REPO` / `GITHUB_BRANCH` 填错。
- `422`：请求内容格式不合法，检查文件名和路径。
- 网络超时：检查网络连通性与 GitHub API 可访问性。

## 7. 模块化开发结构

- `backend/internal/service/path_normalizer.go`
  - 负责标题 slug 化、文件名清洗、标准路径构建。

- `backend/internal/service/github_client.go`
  - 负责 GitHub Contents API 封装（查询 SHA、上传文件、删除文件、列目录）。

- `backend/internal/service/github_sync_service.go`
  - 负责上传/删除流程编排（单文件同步、文件夹同步、GitHub 删除）。

- `backend/internal/handler/skill.go`
  - 负责在上传/删除接口中接入同步服务并写回数据库状态字段。

- `backend/internal/model/skill.go`
  - 保存同步结果：`github_path/github_url/github_sync_status/github_sync_error`。

## 8. 安全建议

- `backend/.env` 不要提交到公共仓库。
- 定期轮换 `GITHUB_TOKEN`。
- 使用最小权限原则，仅授予必要仓库权限。
