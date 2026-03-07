# Skill Hub Keycloak 接入指南（中文，适配 2026 最新版）

> 更新时间：2026-03-07  
> 适配目标：在**不破坏你现有本地 JWT 体系**的前提下，增加企业 SSO（Keycloak）登录。

## 1. 先检查你当前后端（已核对）

你当前项目后端认证现状：

- 本地账号体系：`username + password`（`backend/internal/handler/auth.go`）。
- 鉴权机制：后端自签 JWT（`AuthMiddleware` / `OptionalAuthMiddleware`）。
- 路由：`/api/auth/register`、`/api/auth/login`、`/api/auth/me` 已可用（`backend/cmd/server/main.go`）。
- 现状缺口：还没有 Keycloak 配置项、Keycloak token 校验器、`/api/auth/keycloak/exchange` 交换端点。

结论：推荐采用“**Keycloak token 换本地 JWT**”模式，改动最小、和你现有前后端兼容最好。

---

## 2. 官方版本与关键变化（截至 2026-03-07）

- Keycloak Server 最新：**26.5.5**（官方 Downloads 页）。
- JavaScript Adapter 独立发布：**26.2.3**（官方 Downloads 页）。
- 新版控制台里，不再用旧文档常写的 `Access Type=public/confidential` 描述；应使用：
  - `Client authentication = Off`（表示 Public Client，适合 SPA）
  - `Client authentication = On`（表示 Confidential Client）

---

## 3. 推荐架构（与你当前项目最兼容）

1. 前端用 `keycloak-js` 完成浏览器 OIDC 登录，拿到 Keycloak Access Token。  
2. 前端调用后端：`POST /api/auth/keycloak/exchange`，在 `Authorization: Bearer <token>` 里带上 Keycloak token。  
3. 后端验证 Keycloak token（签名 + issuer + 过期 + 可选 `azp`），校验通过后：
   - 自动创建/绑定本地用户（建议用户名键为 `kc:<sub>`）
   - 签发你现有本地 JWT（继续沿用你当前 `AuthMiddleware`）
4. 后续接口全部继续使用你本地 JWT，无需改业务 API。

---

## 4. Keycloak 控制台配置（最新版术语）

## 4.1 创建 Realm

- Realm：`skill-hub`

## 4.2 创建前端客户端（SPA）

- Client ID：`skill-hub-frontend`
- Client type：`OpenID Connect`
- `Client authentication`：**Off**（Public）
- `Standard flow`：**On**
- `Direct access grants`：建议 Off
- `Valid redirect URIs`：如 `https://your-domain.com/*`（本地可加 `http://localhost:5173/*`）
- `Web origins`：如 `https://your-domain.com`（本地可加 `http://localhost:5173`）

## 4.3（可选）创建后端客户端（仅在你要用 introspection 时）

- Client ID：`skill-hub-backend`
- `Client authentication`：On（Confidential）

说明：你本指南默认采用 JWKS 离线验签，不强依赖 introspection。

---

## 5. 后端改造

## 5.1 配置项（`backend/internal/config/config.go`）

在 `Config` 增加：

```go
KeycloakEnabled          bool
KeycloakIssuer           string // 例如: https://sso.example.com/realms/skill-hub
KeycloakFrontendClientID string // 例如: skill-hub-frontend
```

在 `Load()` 增加：

```go
KeycloakEnabled:          getEnvBool("KEYCLOAK_ENABLED", false),
KeycloakIssuer:           getEnv("KEYCLOAK_ISSUER", ""),
KeycloakFrontendClientID: getEnv("KEYCLOAK_FRONTEND_CLIENT_ID", ""),
```

## 5.2 新建 Keycloak 验签器（建议文件：`backend/internal/handler/keycloak.go`）

依赖（建议）：`github.com/MicahParks/keyfunc/v3` + `github.com/golang-jwt/jwt/v5`

```bash
cd backend
go get github.com/MicahParks/keyfunc/v3
```

示例（关键逻辑）：

```go
package handler

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/MicahParks/keyfunc/v3"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

type KeycloakTokenClaims struct {
    Sub               string
    PreferredUsername string
    Email             string
    Azp               string
}

type KeycloakVerifier struct {
    issuer           string
    frontendClientID string
    jwks             *keyfunc.Keyfunc
}

func NewKeycloakVerifier(ctx context.Context, issuer, frontendClientID string) (*KeycloakVerifier, error) {
    issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
    if issuer == "" {
        return nil, errors.New("keycloak issuer is required")
    }

    jwksURL := fmt.Sprintf("%s/protocol/openid-connect/certs", issuer)
    jwks, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
    if err != nil {
        return nil, err
    }

    return &KeycloakVerifier{
        issuer:           issuer,
        frontendClientID: strings.TrimSpace(frontendClientID),
        jwks:             jwks,
    }, nil
}

func (v *KeycloakVerifier) VerifyToken(raw string) (*KeycloakTokenClaims, error) {
    token, err := jwt.Parse(raw, v.jwks.Keyfunc,
        jwt.WithIssuer(v.issuer),
        jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512"}),
    )
    if err != nil || !token.Valid {
        return nil, errors.New("invalid keycloak token")
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, errors.New("invalid token claims")
    }

    expFloat, ok := claims["exp"].(float64)
    if !ok || time.Now().Unix() >= int64(expFloat) {
        return nil, errors.New("token expired")
    }

    sub, _ := claims["sub"].(string)
    if strings.TrimSpace(sub) == "" {
        return nil, errors.New("sub is required")
    }

    azp, _ := claims["azp"].(string)
    if v.frontendClientID != "" && azp != "" && azp != v.frontendClientID {
        return nil, errors.New("unexpected azp")
    }

    preferredUsername, _ := claims["preferred_username"].(string)
    email, _ := claims["email"].(string)

    return &KeycloakTokenClaims{
        Sub:               sub,
        PreferredUsername: preferredUsername,
        Email:             email,
        Azp:               azp,
    }, nil
}

func (v *KeycloakVerifier) ExchangeAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization bearer token required"})
            c.Abort()
            return
        }

        raw := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
        parsed, err := v.VerifyToken(raw)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid keycloak token"})
            c.Abort()
            return
        }

        c.Set("keycloakClaims", parsed)
        c.Next()
    }
}
```

## 5.3 在 `auth.go` 增加交换端点

新增方法：`ExchangeKeycloakToken`

关键点：

- 从 `c.Get("keycloakClaims")` 读中间件放入的 claims。
- 使用 `kc:<sub>` 作为本地用户唯一键（防止与本地账号重名）。
- 首次登录自动建用户；`Password` 字段请写入随机字符串的 bcrypt 哈希（不要留空明文）。
- 最后调用现有 `generateToken()` 签发你本地 JWT。

示例（简化）：

```go
func (h *AuthHandler) ExchangeKeycloakToken(c *gin.Context) {
    claimsAny, ok := c.Get("keycloakClaims")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "keycloak claims missing"})
        return
    }
    kcClaims, ok := claimsAny.(*KeycloakTokenClaims)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid keycloak claims"})
        return
    }

    localUsername := "kc:" + kcClaims.Sub

    var user model.User
    err := h.db.Where("username = ?", localUsername).First(&user).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        randomPass := uuid.NewString()
        hashed, _ := bcrypt.GenerateFromPassword([]byte(randomPass), bcrypt.DefaultCost)
        user = model.User{
            Username: localUsername,
            Password: string(hashed),
        }
        if err := h.db.Create(&user).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create local user"})
            return
        }
    } else if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load local user"})
        return
    }

    token, err := h.generateToken(user.ID, user.Username)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate local token"})
        return
    }

    c.JSON(http.StatusOK, authResponse{Token: token, User: user})
}
```

该段需要的额外 import：`errors`、`github.com/google/uuid`、`golang.org/x/crypto/bcrypt`、`gorm.io/gorm`、`skill-hub/internal/model`。

## 5.4 在 `main.go` 注册路由

```go
if cfg.KeycloakEnabled {
    verifier, err := handler.NewKeycloakVerifier(context.Background(), cfg.KeycloakIssuer, cfg.KeycloakFrontendClientID)
    if err != nil {
        log.Fatalf("init keycloak verifier failed: %v", err)
    }

    api.POST("/auth/keycloak/exchange",
        verifier.ExchangeAuthMiddleware(),
        authHandler.ExchangeKeycloakToken,
    )
}
```

---

## 6. 前端改造

## 6.1 安装 Keycloak JS 适配器

```bash
cd frontend
npm install keycloak-js
```

## 6.2 新建 Keycloak 配置（`frontend/src/config/keycloak.ts`）

```ts
import Keycloak from 'keycloak-js'

const keycloak = new Keycloak({
  url: import.meta.env.VITE_KEYCLOAK_URL,
  realm: import.meta.env.VITE_KEYCLOAK_REALM,
  clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID,
})

export default keycloak
```

## 6.3 在 `AuthContext` 增加 `loginWithKeycloak`

建议流程：

1. `await keycloak.init({ onLoad: 'check-sso', pkceMethod: 'S256' })`
2. 若未登录则 `await keycloak.login({ redirectUri: window.location.href })`
3. `await keycloak.updateToken(30)` 保证 token 新鲜
4. 调后端交换接口 `/api/auth/keycloak/exchange`
5. 保存后端返回的**本地 JWT**到你现有 `auth_token`

示例（简化）：

```ts
const loginWithKeycloak = async () => {
  await keycloak.init({ onLoad: 'check-sso', pkceMethod: 'S256' })

  if (!keycloak.authenticated) {
    await keycloak.login({ redirectUri: window.location.href })
    return
  }

  await keycloak.updateToken(30)
  if (!keycloak.token) throw new Error('Missing Keycloak token')

  const res = await fetch('/api/auth/keycloak/exchange', {
    method: 'POST',
    headers: { Authorization: `Bearer ${keycloak.token}` },
  })
  if (!res.ok) throw new Error('Keycloak token exchange failed')

  const data = await res.json()
  localStorage.setItem('auth_token', data.token) // 仅保存本地 JWT
  setToken(data.token)
  setUser(data.user)
}
```

## 6.4 登录弹窗加 SSO 按钮

在 `frontend/src/components/LoginModal.tsx` 把原来的 SSO placeholder 改为真实按钮，调用 `loginWithKeycloak()`。

---

## 7. 环境变量

后端 `.env`：

```env
KEYCLOAK_ENABLED=true
KEYCLOAK_ISSUER=https://keycloak.example.com/realms/skill-hub
KEYCLOAK_FRONTEND_CLIENT_ID=skill-hub-frontend
```

前端 `.env`：

```env
VITE_KEYCLOAK_URL=https://keycloak.example.com
VITE_KEYCLOAK_REALM=skill-hub
VITE_KEYCLOAK_CLIENT_ID=skill-hub-frontend
```

---

## 8. 联调验收清单

1. 浏览器点 SSO，跳转 Keycloak 登录页并成功回调。  
2. `/api/auth/keycloak/exchange` 返回本地 JWT + 用户对象。  
3. `localStorage.auth_token` 为后端本地 JWT（不是 Keycloak token）。  
4. `/api/auth/me` 能读出登录用户。  
5. 受保护接口（上传、删除、点赞、收藏等）可正常访问。  
6. 旧用户名密码登录路径不受影响。  

---

## 9. 常见坑（已按新版修正）

- 不要再写旧控制台术语 `Access Type`；新版看 `Client authentication`。  
- 旧文档常见 `keyfunc/v2` 示例已过时，建议用 `keyfunc/v3`。  
- 不建议把 Keycloak token 持久化到 `localStorage`；本项目只落地后端本地 JWT。  
- 如果你改成 Keycloak lightweight/opaque access token，需走 introspection（官方说明：introspection 端点只允许 confidential client 调用）。  

---

## 10. 官方参考链接

- Keycloak Downloads（版本信息）: https://www.keycloak.org/downloads  
- JavaScript Adapter（`onLoad`、`pkceMethod`、安全建议）: https://www.keycloak.org/securing-apps/javascript-adapter  
- OIDC 端点（well-known、certs、introspection）: https://www.keycloak.org/securing-apps/oidc-layers  
- keyfunc v3（Go JWKS 验签库）: https://github.com/MicahParks/keyfunc  
