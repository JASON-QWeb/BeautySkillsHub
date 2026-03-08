package handler

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skill-hub/internal/model"
	"skill-hub/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db        *gorm.DB
	secret    []byte
	avatarDir string
}

func NewAuthHandler(db *gorm.DB, avatarDir string) *AuthHandler {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		appEnv := strings.TrimSpace(strings.ToLower(os.Getenv("APP_ENV")))
		if appEnv != "" && appEnv != "local" {
			panic("JWT_SECRET must be set outside local environment")
		}
		secret = generateEphemeralJWTSecret()
		slog.Warn("JWT_SECRET 未配置，已使用临时随机密钥", "impact", "重启后 token 将失效")
	}
	return &AuthHandler{db: db, secret: []byte(secret), avatarDir: avatarDir}
}

func generateEphemeralJWTSecret() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		// fallback should still avoid hard-coded shared secrets
		return base64.RawURLEncoding.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type authResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username (min 2 chars) and password (min 6 chars) are required"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if len([]rune(req.Username)) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username (min 2 chars) and password (min 6 chars) are required"})
		return
	}

	// Check if user exists
	var existing model.User
	existsQuery := h.db.Select("id").Where("username = ?", req.Username).Limit(1).Find(&existing)
	if existsQuery.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate username"})
		return
	}
	if existsQuery.RowsAffected > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := model.User{
		Username: req.Username,
		Password: string(hashed),
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate pixel avatar
	if avatarFile, err := service.GenerateAvatar(user.Username, h.avatarDir); err == nil {
		user.AvatarURL = "/api/avatars/" + avatarFile
		h.db.Model(&user).Update("avatar_url", user.AvatarURL)
	} else {
		slog.Warn("failed to generate avatar", "username", user.Username, "error", err)
	}

	token, err := h.generateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, authResponse{Token: token, User: user})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	var user model.User
	if err := h.db.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	token, err := h.generateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, authResponse{Token: token, User: user})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) generateToken(userID uint, username string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.secret)
}

// ServeAvatar serves avatar image files.
func (h *AuthHandler) ServeAvatar(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(h.avatarDir, filepath.Base(filename))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Avatar not found"})
		return
	}
	c.File(filePath)
}

// AuthMiddleware returns a Gin middleware that validates JWT tokens.
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		userID, username, err := h.parseToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		c.Set("userID", userID)
		c.Set("username", username)
		c.Next()
	}
}

// OptionalAuthMiddleware parses a bearer token when present, without rejecting public requests.
func (h *AuthHandler) OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		userID, username, err := h.parseToken(tokenStr)
		if err == nil {
			c.Set("userID", userID)
			c.Set("username", username)
		}
		c.Next()
	}
}

func (h *AuthHandler) parseToken(tokenStr string) (uint, string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.secret, nil
	})
	if err != nil || !token.Valid {
		return 0, "", jwt.ErrTokenMalformed
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", jwt.ErrTokenMalformed
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, "", jwt.ErrTokenMalformed
	}
	username, _ := claims["username"].(string)
	return uint(userIDFloat), username, nil
}
