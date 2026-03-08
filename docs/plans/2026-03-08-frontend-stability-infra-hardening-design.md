# Frontend Stability And Infrastructure Hardening Design

## Goal

在一个隔离 worktree 中同时完成以下两类修复：

- 前端稳定性与可维护性问题：`ErrorBoundary`、会话过期处理、README 缓存限额、可取消请求、共享鉴权请求头、Profile 分页、AI 角色鼠标跟随节流、Dialog `Escape` 关闭。
- 后端基础设施问题：安全数据库连接校验、`/health` 健康检查、非本地环境强制 `JWT_SECRET`、Docker 非 root 运行。

本次设计目标不是“最小修补”，而是把请求层、会话生命周期、Profile 数据来源和基础设施启动约束一起收敛到更稳的生产形态。

## Current Problems

### Frontend

- 根组件缺少 `ErrorBoundary`，任何 render 抛错都可能导致整棵 React 树白屏。
- 鉴权状态只在 `AuthProvider` 初始化时调用一次 `/auth/me` 验证；token 后续过期时，前端没有统一收到 `401` 后自动登出。
- API 请求层没有统一封装，`getAuthHeaders()` 在多个文件重复，`fetch` 也没有标准化的 `AbortSignal` 透传。
- `SkillDetailPage` 的 `readmeGlobalCache` 是无限增长的模块级 `Map`。
- `ProfilePage` 拉取的是“全站前 500 条资源”，再在前端过滤出当前用户上传项，数据量大时既慢又不准确。
- `AIChatCharacter` 直接在 `mousemove` 上高频 setState。
- `DialogContext` 只能点击关闭，不支持键盘 `Escape`。

### Backend

- `DATABASE_URL` 默认值包含 `sslmode=disable`，生产漏配时会退回不安全连接。
- 服务没有内置健康检查端点，容器/编排层无法标准化探测。
- `JWT_SECRET` 缺失时自动生成临时密钥，重启后所有 token 失效，生产不可接受。
- Docker 镜像默认 root 运行。

## Design Decisions

### 1. Shared Frontend Request Layer

新增统一请求模块，负责：

- 读取本地 token
- 生成鉴权请求头
- 对外暴露统一 `apiFetch()` / `apiRequest()` 风格方法
- 支持 `AbortSignal`
- 在收到鉴权请求的 `401` 时触发全局 unauthorized handler

`AuthProvider` 负责注册 unauthorized handler，并在 token 明显过期或接口返回 `401` 时清空登录态。

这样可以一次性解决：

- token 过期后 UI 仍认为已登录
- `getAuthHeaders()` 重复定义
- effect 请求没有标准 signal 入口

### 2. Token Lifecycle Handling

前端不做 refresh token 体系扩展，本次只做两件事：

- 启动时解析 JWT `exp`，已过期则立即登出，不再等待一次失败请求。
- 任何带鉴权的 API 请求返回 `401` 时，统一清理本地会话。

这是当前项目能在不引入新认证协议的前提下最稳的处理方式。

### 3. Global Error Boundary

新增应用级 `ErrorBoundary`，包裹主应用树，并提供：

- 简单错误提示
- 返回首页/重新加载动作
- 开发时输出错误详情，生产时避免直接把原始栈渲染到 UI

它的目标是把“整页白屏”降级成“局部 fallback”。

### 4. Bounded README Cache

把 `readmeGlobalCache` 从裸 `Map` 抽成小型 LRU 缓存模块，使用“最多保留固定条目数”的策略。

本次采用：

- 以 `skillId` 为 key
- `maxEntries = 50`
- 命中时刷新最近访问顺序
- 超限时淘汰最老条目

不引入 TTL，避免增加时间相关复杂度；问题的核心是无限增长，不是跨天陈旧数据。

### 5. Abortable Effects

所有由组件生命周期驱动的远程请求统一改成：

- 在 `useEffect` 中创建 `AbortController`
- 把 `signal` 传入 API 层
- cleanup 时 `abort()`
- 忽略 `AbortError`

事件驱动型请求（如点击点赞、收藏、删除）不强制一律做 controller 管理，但 API 层同样支持 `signal`，便于后续扩展。

### 6. Profile Pagination And Stats

不再让 `ProfilePage` 依赖“全站资源列表 + 前端过滤”。

新增后端 `me` 维度接口，返回：

- 当前用户上传资源分页列表
- 上传统计：`total_items`、`total_downloads`、`total_likes`
- 最近活动：最近发布 / 最近复核的条目

前端 `ProfilePage` 改为只拉自己的分页数据和活动摘要。这样既解决一次性拉 500 条，也让统计逻辑不再依赖首页列表的可见性规则。

### 7. AI Character Mouse Tracking

把 `mousemove -> setState` 改成 `requestAnimationFrame` 合并更新：

- 原生事件只记录最后一次坐标
- 每帧最多触发一次 React state 更新

这能把高频鼠标事件收敛到浏览器绘制节奏。

### 8. Dialog Escape Support

在 `DialogContext` 中，当 dialog 打开时监听 `keydown`：

- `Escape` 关闭当前 dialog
- confirm 按取消语义处理
- alert 按确认关闭语义处理

这样不会改变现有点击行为，只补齐键盘交互。

### 9. Backend Runtime Hardening

后端新增一个集中校验步骤，在启动时区分 `local` 与非 `local`：

- `DATABASE_URL` 在非本地环境不能缺失
- 非本地环境下不能使用 `sslmode=disable`
- 非本地环境必须显式配置 `JWT_SECRET`

同时新增：

- `/health`：返回进程存活状态，并带数据库 ping 结果
- Docker 非 root 用户

## Testing Strategy

### Frontend

- 为 README LRU 缓存、JWT 过期解析、API 请求层 unauthorized 处理、AI mouse scheduling 等抽离纯逻辑测试。
- 为 Profile 数据转换保留轻量纯函数测试。
- 继续使用 `node --test`。
- 最终用 `npm run build` 做编译与集成校验。

### Backend

- 为配置校验、`JWT_SECRET` 约束、`/health` 路由补 Go 测试。
- 现有后端全量测试必须继续通过。

### Infra

- 通过 Dockerfile 内容变更和镜像构建约束验证非 root 运行。
- 若本地环境缺少 Docker CLI，则至少保证 Dockerfile 静态正确并记录未执行项。

## Files Expected To Change

### Frontend

- `frontend/src/App.tsx`
- `frontend/src/contexts/AuthContext.tsx`
- `frontend/src/contexts/DialogContext.tsx`
- `frontend/src/components/AIChatCharacter.tsx`
- `frontend/src/features/skill-detail/SkillDetailPage.tsx`
- `frontend/src/features/profile/ProfilePage.tsx`
- `frontend/src/services/api/skills.ts`
- `frontend/src/services/api/content-assets.ts`
- `frontend/src/services/api/ai.ts`
- `frontend/src/services/api/client.ts`
- new helper files under `frontend/src/services/api/` and `frontend/src/features/skill-detail/`

### Backend

- `backend/internal/config/config.go`
- `backend/internal/config/config_test.go`
- `backend/internal/handler/auth.go`
- `backend/internal/handler/auth_test.go`
- `backend/cmd/server/main.go`
- `backend/cmd/server/security_test.go`
- new health-related handler/router helpers if needed
- `backend/Dockerfile`
- `frontend/Dockerfile`

### Docs

- `docs/review_fix/<date>-frontend-and-infra-review-fixes.md`
- `docs/plans/<date>-frontend-stability-infra-hardening-implementation-plan.md`

## Risks

- Profile 改成 `me` 维度接口后，前端页面需要同时适配新响应结构和分页状态。
- 统一 unauthorized handler 需要避免把登录/注册等匿名请求误判成会话失效。
- `ErrorBoundary` 只能捕获 render/lifecycle 错误，不能替代事件处理和异步错误处理。
- 非本地环境强制 `JWT_SECRET` 和安全 DB URL 会改变部署要求，需要在文档中明确说明。
