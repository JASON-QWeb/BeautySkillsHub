# 2026-03-08 Frontend And Infrastructure Review Fixes

## Scope

本轮修复覆盖两组问题：

- 前端稳定性与会话管理
- 基础设施与运行时安全

对应 review 中的以下项：

- 无 `ErrorBoundary`
- token 失效后仍保留登录态
- `readmeGlobalCache` 无大小限制
- 生命周期请求缺少 `AbortController`
- `getAuthHeaders()` 重复定义
- `ProfilePage` 一次性拉取 500 条数据
- `AIChatCharacter` `mousemove` 无节流
- Dialog 不支持 `Escape`
- `DATABASE_URL` 默认可落到 `sslmode=disable`
- Docker 容器以 root 运行
- 缺少 `/health`
- `JWT_SECRET` 未配置时使用临时密钥
- 后端日志缺少结构化/分级

## Frontend Fixes

### 1. 全局渲染兜底

- 新增 [frontend/src/components/AppErrorBoundary.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/components/AppErrorBoundary.tsx)
- 在 [frontend/src/App.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/App.tsx) 顶层包裹整棵应用树
- 组件渲染异常时不再整站白屏，用户可返回首页或刷新

### 2. token 过期与 401 会话失效处理

- 新增共享请求层 [frontend/src/services/api/request.ts](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/services/api/request.ts)
- 在请求前解析 JWT `exp`，启动时发现过期 token 会立刻退出登录
- 对带鉴权的 API 请求统一接入 `401 -> logout()` 处理
- [frontend/src/contexts/AuthContext.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/contexts/AuthContext.tsx) 已改为使用共享请求层

### 3. README 缓存加上限

- 新增 [frontend/src/features/skill-detail/readmeCache.ts](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/features/skill-detail/readmeCache.ts)
- [frontend/src/features/skill-detail/SkillDetailPage.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/features/skill-detail/SkillDetailPage.tsx) 现在使用固定上限缓存
- 缓存达到上限时淘汰最旧条目，避免长期浏览后内存持续增长

### 4. 生命周期请求统一支持取消

已为以下典型页面和组件补上 `AbortController`：

- [frontend/src/contexts/AuthContext.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/contexts/AuthContext.tsx)
- [frontend/src/features/home/HomePage.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/features/home/HomePage.tsx)
- [frontend/src/components/RightSidebar.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/components/RightSidebar.tsx)
- [frontend/src/components/TrendingList.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/components/TrendingList.tsx)
- [frontend/src/features/profile/ProfilePage.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/features/profile/ProfilePage.tsx)
- [frontend/src/features/review/ReviewPage.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/features/review/ReviewPage.tsx)
- [frontend/src/features/skill-detail/SkillDetailPage.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/features/skill-detail/SkillDetailPage.tsx)
- [frontend/src/components/SkillsInstallModal.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/components/SkillsInstallModal.tsx)
- 4 个上传编辑页

说明：

- 当前仍保留的 `fetch(url)` 与 `let cancelled = false` 只用于本地 `File` / `ObjectURL` / 图片裁剪预览，不是后端 API 请求

### 5. 共享鉴权头，移除重复实现

- `getAuthHeaders()` 现在只保留在 [frontend/src/services/api/request.ts](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/services/api/request.ts)
- [frontend/src/services/api/skills.ts](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/services/api/skills.ts)、[frontend/src/services/api/content-assets.ts](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/services/api/content-assets.ts)、[frontend/src/services/api/ai.ts](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/services/api/ai.ts) 已切到共享请求层

### 6. Profile 改为服务端分页

- 新增后端接口 `GET /api/me/uploads`
- 新增 [backend/internal/service/profile.go](/Users/qianjianghao/Desktop/Skill_Hub/backend/internal/service/profile.go)
- 新增 [backend/internal/handler/profile_handlers.go](/Users/qianjianghao/Desktop/Skill_Hub/backend/internal/handler/profile_handlers.go)
- [frontend/src/features/profile/ProfilePage.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/features/profile/ProfilePage.tsx) 不再抓全站 500 条后前端过滤，而是直接请求“当前用户上传 + 统计 + 活动”

### 7. 小型交互修复

- [frontend/src/components/AIChatCharacter.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/components/AIChatCharacter.tsx) 已把 `mousemove` 改为 RAF 合并调度
- [frontend/src/contexts/DialogContext.tsx](/Users/qianjianghao/Desktop/Skill_Hub/frontend/src/contexts/DialogContext.tsx) 现支持 `Escape` 关闭

## Infrastructure Fixes

### 1. 生产环境数据库连接安全校验

- [backend/internal/config/config.go](/Users/qianjianghao/Desktop/Skill_Hub/backend/internal/config/config.go) 新增运行时校验
- 非 `local` 环境下：
  - `DATABASE_URL` 不能为空
  - `DATABASE_URL` 不能使用 `sslmode=disable`

### 2. `/health` 健康检查

- 新增 [backend/cmd/server/health.go](/Users/qianjianghao/Desktop/Skill_Hub/backend/cmd/server/health.go)
- 启动链路在 [backend/cmd/server/main.go](/Users/qianjianghao/Desktop/Skill_Hub/backend/cmd/server/main.go) 注册 `/health`
- 健康检查会调用数据库 `Ping`
- `docker-compose.yml` 的 backend healthcheck 已切换到 `/health`

### 3. JWT_SECRET 强制要求

- [backend/internal/handler/auth.go](/Users/qianjianghao/Desktop/Skill_Hub/backend/internal/handler/auth.go)
- 非 `local` 环境如果未设置 `JWT_SECRET`，服务会直接 panic 启动失败
- `local` 环境仍允许临时随机密钥，保留本地开发便利性

### 4. Docker 非 root 运行

- [backend/Dockerfile](/Users/qianjianghao/Desktop/Skill_Hub/backend/Dockerfile) 新增 `skillhub` 用户并切换 `USER skillhub`
- [frontend/Dockerfile](/Users/qianjianghao/Desktop/Skill_Hub/frontend/Dockerfile) 切换为 `USER nginx`
- [frontend/nginx.conf](/Users/qianjianghao/Desktop/Skill_Hub/frontend/nginx.conf) 监听端口从 `80` 改到 `8080`
- [docker-compose.yml](/Users/qianjianghao/Desktop/Skill_Hub/docker-compose.yml) 同步把前端容器内端口映射改为 `8080`
- [DEPLOYMENT.md](/Users/qianjianghao/Desktop/Skill_Hub/DEPLOYMENT.md) 已同步更新 `docker run` 示例

### 5. 结构化日志

- 新增 [backend/internal/logging/logging.go](/Users/qianjianghao/Desktop/Skill_Hub/backend/internal/logging/logging.go)
- `local/test` 环境使用文本日志，便于本地阅读
- 非本地环境使用 `JSON slog`，便于接日志平台
- `cmd/server`、`cmd/migrate`、`cmd/clear-db` 以及关键 handler/middleware/service 已从 `log.Printf/Fatalf` 切到分级结构化日志

## Verification

本轮实现后应至少通过：

- `node --test frontend/src/services/api/request.test.ts frontend/src/features/skill-detail/readmeCache.test.ts frontend/src/contexts/dialogKeydown.test.ts frontend/src/components/aiMouseTracking.test.ts frontend/docker-runtime.test.mjs`
- `cd backend && go test ./...`
- `cd frontend && npm run build`
- `git diff --check`

如本地安装了 Docker，还应补跑：

- `docker build -f backend/Dockerfile backend`
- `docker build -f frontend/Dockerfile frontend`
