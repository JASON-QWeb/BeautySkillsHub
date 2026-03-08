# Skill Hub Architecture Review

> External review received on 2026-03-08
> Internally re-verified against `main` on 2026-03-08

---

## 使用说明

这份文档记录的是“原始 review 条目在当前代码里的真实状态”，不是修复过程文档。

修复过程和实施细节见：

- [docs/review_fix/2026-03-08-architecture-review.md](../review_fix/2026-03-08-architecture-review.md)
- [docs/review_fix/2026-03-08-backend-review-round2.md](../review_fix/2026-03-08-backend-review-round2.md)
- [docs/review_fix/2026-03-08-frontend-and-infra-review-fixes.md](../review_fix/2026-03-08-frontend-and-infra-review-fixes.md)

状态说明：

- `已修复`：问题已在主线代码中解决
- `待修复`：问题成立，但还没处理
- `未采纳为缺陷`：原 review 表述过头，内部复核后不作为当前缺陷处理
- `待评估`：需要结合真实数据量或未来功能边界再判断

---

## 1. Backend

### 1.1 CORS 配置为通配符

- 状态：`已修复`
- 现状：后端已改为 allowlist，不再返回 `Access-Control-Allow-Origin: *`

### 1.2 缺少安全响应头

- 状态：`已修复`
- 现状：已接入统一安全头中间件

### 1.3 无速率限制

- 状态：`已修复`
- 现状：登录、注册、审核重试、AI chat 已有限流

### 1.4 `skill_likes` / `skill_favorites` 缺少外键约束

- 状态：`已修复`
- 现状：已添加外键和 `ON DELETE CASCADE`

### 1.5 文件上传 error path 中 `f.Close()` 未用 defer

- 状态：`未采纳为缺陷`
- 复核结论：当前代码是手动关闭，不是已证实的资源泄漏；它更像可维护性问题，而不是当前高优先级 bug

### 1.6 `IncrementDownload` 错误被静默忽略

- 状态：`已修复`
- 现状：下载计数失败会记录并区分处理，不再静默吞掉

### 1.7 `validateSourceURL` 不阻止内网 IP

- 状态：`待修复`
- 说明：当前后端主要是存储 URL，不是主动抓取；但从防御式设计角度看，仍建议补私网地址拦截

### 1.8 `LikeSkill` 计数竞态

- 状态：`已修复`
- 现状：返回计数已改为同事务内读取

### 1.9 上传字段无最大长度校验

- 状态：`已修复`
- 现状：`name`、`description`、`tags`、`author`、`source_url` 已有限制

### 1.10 常用查询缺少复合索引

- 状态：`待评估`
- 复核结论：这是性能优化方向，不应在没有 `EXPLAIN ANALYZE` 的情况下直接认定为 bug

### 1.11 `JWT_SECRET` 未配置时静默使用临时密钥

- 状态：`已修复`
- 现状：非 `local` 环境未配置会直接 fail-fast

### 1.12 数据库连接默认 `sslmode=disable`

- 状态：`已修复`
- 现状：非 `local` 环境禁止 `sslmode=disable`

---

## 2. Frontend

### 2.1 无 ErrorBoundary

- 状态：`已修复`

### 2.2 Token 过期无处理

- 状态：`已修复`
- 现状：401 会统一触发前端退出登录

### 2.3 `readmeGlobalCache` 无大小限制

- 状态：`已修复`
- 现状：已改为有限容量缓存

### 2.4 fetch 缺少 `AbortController`

- 状态：`已修复`
- 现状：主要页面和生命周期请求已补齐取消逻辑

### 2.5 `getAuthHeaders()` 重复定义

- 状态：`已修复`

### 2.6 `ProfilePage` 一次加载 500 条

- 状态：`已修复`
- 现状：已切到服务端分页接口 `/api/me/uploads`

### 2.7 DialogContext 竞态

- 状态：`待修复`
- 说明：当前 `showAlert` / `showConfirm` 仍是单状态覆盖模型，连续触发时存在队列化不足的问题

### 2.8 `AIChatCharacter` mousemove 无节流

- 状态：`已修复`

### 2.9 Dialog 不支持 Escape

- 状态：`已修复`

### 2.10 `dangerouslySetInnerHTML` 额外消毒

- 状态：`待修复`
- 复核结论：当前主线没有接入 `DOMPurify`，原 review 中“已修复”这一说法不成立

---

## 3. Infrastructure

### 3.1 Docker 容器以 root 运行

- 状态：`已修复`
- 现状：backend 以 `skillhub` 用户运行，frontend 使用 `nginxinc/nginx-unprivileged:alpine`

### 3.2 无健康检查端点

- 状态：`已修复`
- 现状：后端健康检查路由是 `/health`，不是 `/api/health`

### 3.3 Docker 镜像版本未完全锁定

- 状态：`待评估`
- 复核结论：当前 `postgres:16-alpine` / `redis:7-alpine` 仍是大版本锁定，不是完整 patch pin

### 3.4 标准 log、无结构化分级

- 状态：`已修复`
- 现状：已迁移到 `log/slog`

---

## 4. 当前结论

### 已修复

- CORS allowlist
- 安全响应头
- 限流
- engagement 外键约束
- 下载计数错误处理
- 点赞计数竞态
- 上传字段长度校验
- 生产环境 `JWT_SECRET` 约束
- 非本地环境安全数据库连接约束
- ErrorBoundary
- token 过期 / 401 处理
- README 缓存上限
- 主要 fetch 请求取消
- `getAuthHeaders` 去重
- Profile 上传分页
- AI 角色鼠标调度
- Dialog Escape
- 非 root Docker
- `/health`
- 结构化日志

### 待修复

1. `validateSourceURL` 私网地址拦截
2. DialogContext 队列化
3. `dangerouslySetInnerHTML` 额外消毒策略

### 不应直接按原 review 认定

1. 文件上传 error path 资源泄漏
2. 复合索引缺失一定是 bug
3. Docker 镜像“已完全锁 patch 版本”
