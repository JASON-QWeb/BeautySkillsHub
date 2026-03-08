# 2026-03-08 Architecture Review Fixes

对应 review: [docs/review/2026-03-08-architecture-review.md](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/docs/review/2026-03-08-architecture-review.md)

## 本次修复范围

本次修复覆盖以下 review 条目：

- 1.1 `CRITICAL - CORS 配置为通配符`
- 1.2 `HIGH - 缺少安全响应头`
- 1.3 `HIGH - 无速率限制`

不在本次修复范围内的条目仍需后续单独处理，例如外键约束、上传 error path 资源释放、SSRF 防护等。

## 根因

### 1.1 CORS

后端此前在全局中间件中直接返回 `Access-Control-Allow-Origin: *`，没有基于部署环境区分可信来源，也没有对预检请求进行精确匹配。

### 1.2 安全响应头

服务端仅依赖 `gin.Default()` 和自定义 CORS，中间件链没有统一输出安全头，导致点击劫持、MIME sniffing、Referrer 泄露和 HSTS 等基础控制缺失。

### 1.3 速率限制

登录、注册、AI 审核重试和 AI chat 路由没有任何时间窗口控制。虽然审核重试在业务层存在最大次数限制，但缺少网络层节流，无法应对短时间暴力请求和成本滥用。

## 修复内容

### 代码变更

- [backend/internal/config/config.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/internal/config/config.go)
  - 新增 CORS、安全头、HSTS、限流相关配置项和默认值。
- [backend/internal/middleware/cors.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/internal/middleware/cors.go)
  - 将通配符 CORS 改为 allowlist。
  - 支持同源放行、精确 origin 反射、`Vary` 头、预检请求处理和非法跨域 `403`。
- [backend/internal/middleware/security_headers.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/internal/middleware/security_headers.go)
  - 新增统一安全头中间件。
  - 覆盖 `X-Frame-Options`、`X-Content-Type-Options`、`Referrer-Policy`、`Permissions-Policy`、`Cross-Origin-Opener-Policy`、`Content-Security-Policy` 和条件式 `Strict-Transport-Security`。
- [backend/internal/middleware/rate_limit.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/internal/middleware/rate_limit.go)
  - 新增 Redis 优先、内存回退的令牌桶限流实现。
  - 支持按策略名隔离桶、按用户或 IP 识别身份，并返回 `429` 与 `Retry-After`。
- [backend/cmd/server/main.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/cmd/server/main.go)
  - 全局接入安全头和 CORS。
  - 为 `/api/auth/register`、`/api/auth/login`、`/api/skills/:id/review/retry`、`/api/rules/:id/review/retry`、`/api/ai/chat` 接入限流。
- [backend/cmd/server/security.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/cmd/server/security.go)
  - 抽出服务启动所需的安全配置适配和路由限流工厂，便于测试与复用。

### 测试变更

- [backend/internal/middleware/cors_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/internal/middleware/cors_test.go)
- [backend/internal/middleware/security_headers_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/internal/middleware/security_headers_test.go)
- [backend/internal/middleware/rate_limit_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/internal/middleware/rate_limit_test.go)
- [backend/internal/config/config_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/internal/config/config_test.go)
- [backend/cmd/server/security_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/security-hardening-production/backend/cmd/server/security_test.go)

## 新增配置项

### CORS

- `CORS_ALLOWED_ORIGINS`
- `CORS_ALLOWED_METHODS`
- `CORS_ALLOWED_HEADERS`
- `CORS_EXPOSED_HEADERS`
- `CORS_MAX_AGE`

本地开发默认允许常见 localhost / 127.0.0.1 端口；生产环境建议显式设置允许的前端域名列表。

### 安全头

- `SECURITY_CSP`
- `SECURITY_CSP_REPORT_ONLY`
- `HSTS_ENABLED`
- `HSTS_MAX_AGE`
- `HSTS_INCLUDE_SUBDOMAINS`
- `HSTS_PRELOAD`

建议生产环境先在预发布环境验证 `SECURITY_CSP`，如果前端资源源站较复杂，可先启用 `SECURITY_CSP_REPORT_ONLY=true` 观察再收紧。

### 速率限制

- `RATE_LIMIT_LOGIN_CAPACITY`
- `RATE_LIMIT_LOGIN_WINDOW`
- `RATE_LIMIT_REGISTER_CAPACITY`
- `RATE_LIMIT_REGISTER_WINDOW`
- `RATE_LIMIT_REVIEW_RETRY_CAPACITY`
- `RATE_LIMIT_REVIEW_RETRY_WINDOW`
- `RATE_LIMIT_AI_CHAT_CAPACITY`
- `RATE_LIMIT_AI_CHAT_WINDOW`

如果配置了 `REDIS_ADDR`，限流会优先使用 Redis；否则自动回退为单机内存限流。

## 默认策略

- 登录：`5 / 1m / IP`
- 注册：`3 / 30m / IP`
- 审核重试：`3 / 10m / userID 或 IP`
- AI chat：`20 / 1m / userID 或 IP`

这些默认值偏保守，适合作为生产初始值。上线后应基于真实流量和误伤率继续调优。

## 验证命令

本次修复使用以下命令验证：

```bash
cd backend && go test ./internal/config -run 'TestLoad_GitHubDefaults|TestLoad_SecurityOverrides' -v
cd backend && go test ./internal/middleware -v
cd backend && go test ./cmd/server -v
cd backend && go test ./...
cd frontend && npm run build
git diff --check
```

## 剩余运维建议

- 生产部署时必须显式设置 `CORS_ALLOWED_ORIGINS`，不要依赖本地默认值。
- 如果服务运行在反向代理之后，应确认真实客户端 IP 的透传和 Gin 的代理信任配置，否则基于 IP 的限流精度会下降。
- Redis 作为限流后端时，建议纳入监控和告警；当前实现已经在 Redis 不可用时回退到内存限流，但多实例下一致性会下降。
- `Content-Security-Policy` 是本次最容易引发兼容性问题的部分，上线前建议在 staging 先验证一轮前端资源加载。
