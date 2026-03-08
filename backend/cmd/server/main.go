package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"skill-hub/internal/config"
	"skill-hub/internal/handler"
	"skill-hub/internal/middleware"
	"skill-hub/internal/service"
	"skill-hub/internal/service/ai"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect database
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// Initialize services
	skillSvc := service.NewSkillService(db)
	aiSvc := ai.NewService(cfg)
	githubSyncSvc := service.NewGitHubSyncService(cfg, nil)
	var skillContextCache service.SkillContextCache

	if cfg.RedisAddr != "" {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := redisClient.Ping(pingCtx).Err(); err != nil {
			log.Printf("⚠️  Redis 不可用，将回退数据库查询: %v", err)
		} else {
			skillContextCache = service.NewRedisSkillContextCache(redisClient)
			log.Printf("✅ Redis 缓存已启用: %s (db=%d)", cfg.RedisAddr, cfg.RedisDB)
		}
		cancel()
	} else {
		log.Println("ℹ️  Redis 未配置，AI 上下文将回退数据库查询")
	}

	skillContextProvider := service.NewSkillContextProvider(
		skillSvc,
		skillContextCache,
		cfg.AISkillsCacheKey,
		cfg.AISkillsInvalidateChannel,
	)
	if err := skillContextProvider.RefreshSkillsContext(context.Background()); err != nil {
		log.Printf("⚠️  启动预热 AI skills 上下文失败: %v", err)
	}

	// Initialize handlers
	skillHandler := handler.NewSkillHandler(skillSvc, aiSvc, githubSyncSvc, skillContextProvider, cfg)
	contentAssetHandler := handler.NewContentAssetHandler(cfg.UploadDir)
	mcpHandler := handler.NewResourceHandler(skillSvc, "mcp", cfg)
	toolsHandler := handler.NewResourceHandler(skillSvc, "tools", cfg)
	rulesHandler := handler.NewResourceHandler(skillSvc, "rules", cfg)
	avatarDir := "./avatars"
	authHandler := handler.NewAuthHandler(db, avatarDir)

	// Setup Gin
	r := gin.Default()
	r.MaxMultipartMemory = handler.MultipartFormMemoryLimit
	r.Use(middleware.CORS())

	uploadLimitMiddleware := middleware.LimitMultipartBody(handler.MaxUploadRequestBodySize, handler.MultipartFormMemoryLimit)
	contentAssetLimitMiddleware := middleware.LimitMultipartBody(handler.MaxContentAssetRequestBodySize, handler.MultipartFormMemoryLimit)

	// API routes
	api := r.Group("/api")
	{
		// Auth endpoints
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.GET("/auth/me", authHandler.AuthMiddleware(), authHandler.GetMe)

		// Skill/Resource endpoints (public reads + optional auth context)
		publicReads := api.Group("")
		publicReads.Use(authHandler.OptionalAuthMiddleware())
		publicReads.GET("/skills", skillHandler.ListSkills)
		publicReads.GET("/skills/summary", skillHandler.GetSkillSummary)
		publicReads.GET("/skills/trending", skillHandler.GetTrending)
		publicReads.GET("/skills/install-config", skillHandler.GetSkillInstallConfig)
		publicReads.GET("/skills/:id", skillHandler.GetSkill)
			publicReads.GET("/skills/:id/readme", skillHandler.GetSkillReadme)
			publicReads.GET("/skills/:id/download", skillHandler.DownloadSkill)
			publicReads.POST("/skills/:id/download-hit", skillHandler.TrackDownloadHit)
			// Protected: upload, update & delete require auth
			api.POST("/skills", authHandler.AuthMiddleware(), uploadLimitMiddleware, skillHandler.UploadSkill)
			api.GET("/skills/:id/review-status", authHandler.AuthMiddleware(), skillHandler.GetSkillReviewStatus)
			api.POST("/skills/:id/review/retry", authHandler.AuthMiddleware(), skillHandler.RetrySkillReview)
			api.PUT("/skills/:id", authHandler.AuthMiddleware(), skillHandler.UpdateSkill)
		api.DELETE("/skills/:id/stream-delete", authHandler.AuthMiddleware(), skillHandler.StreamDeleteSkill)
		api.DELETE("/skills/:id", authHandler.AuthMiddleware(), skillHandler.DeleteSkill)
		api.POST("/skills/:id/human-review", authHandler.AuthMiddleware(), skillHandler.HumanReviewSkill)
		api.POST("/skills/:id/like", authHandler.AuthMiddleware(), skillHandler.LikeSkill)
		api.DELETE("/skills/:id/like", authHandler.AuthMiddleware(), skillHandler.UnlikeSkill)
		api.POST("/skills/:id/favorite", authHandler.AuthMiddleware(), skillHandler.AddFavoriteSkill)
		api.DELETE("/skills/:id/favorite", authHandler.AuthMiddleware(), skillHandler.RemoveFavoriteSkill)
		api.GET("/me/favorites", authHandler.AuthMiddleware(), skillHandler.ListMyFavorites)

		// Categories
		api.GET("/categories", skillHandler.GetCategories)

			// Thumbnail serving
			api.GET("/thumbnails/:filename", skillHandler.ServeThumbnail)
			api.GET("/content-assets/:filename", contentAssetHandler.ServeImage)
			api.POST("/content-assets/images", authHandler.AuthMiddleware(), contentAssetLimitMiddleware, contentAssetHandler.UploadImage)

		// Avatar serving
		api.GET("/avatars/:filename", authHandler.ServeAvatar)

		// AI Chat
		api.POST("/ai/chat", skillHandler.ChatRecommend)

			// MCP resource routes
			handler.RegisterResourceRoutes(api, mcpHandler, authHandler.AuthMiddleware(), authHandler.OptionalAuthMiddleware(), uploadLimitMiddleware, "/mcps")

			// Tools resource routes
			handler.RegisterResourceRoutes(api, toolsHandler, authHandler.AuthMiddleware(), authHandler.OptionalAuthMiddleware(), uploadLimitMiddleware, "/tools")

		// Rules resource routes (rules require AI + human review)
		rulesPublic := api.Group("/rules")
		rulesPublic.Use(authHandler.OptionalAuthMiddleware())
		rulesPublic.GET("", rulesHandler.List)
		rulesPublic.GET("/summary", rulesHandler.GetSummary)
		rulesPublic.GET("/trending", rulesHandler.GetTrending)
		rulesPublic.GET("/categories", rulesHandler.GetCategories)
		rulesPublic.GET("/:id", rulesHandler.Get)
		rulesPublic.GET("/:id/readme", rulesHandler.GetReadme)
		rulesPublic.GET("/:id/download", rulesHandler.Download)
		rulesPublic.POST("/:id/download-hit", rulesHandler.TrackDownloadHit)

			rulesProtected := api.Group("/rules")
			rulesProtected.Use(authHandler.AuthMiddleware())
			rulesProtected.POST("", uploadLimitMiddleware, skillHandler.UploadSkill)
			rulesProtected.GET("/:id/review-status", skillHandler.GetSkillReviewStatus)
			rulesProtected.POST("/:id/review/retry", skillHandler.RetrySkillReview)
			rulesProtected.POST("/:id/human-review", skillHandler.HumanReviewSkill)
			rulesProtected.PUT("/:id", uploadLimitMiddleware, rulesHandler.Update)
			rulesProtected.DELETE("/:id", rulesHandler.Delete)
			rulesProtected.POST("/:id/like", rulesHandler.Like)
			rulesProtected.DELETE("/:id/like", rulesHandler.Unlike)
		rulesProtected.POST("/:id/favorite", rulesHandler.AddFavorite)
		rulesProtected.DELETE("/:id/favorite", rulesHandler.RemoveFavorite)
	}

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("🚀 Skill Hub 后端启动在 http://localhost%s", addr)
	if cfg.OpenAIKey != "" {
		log.Println("✅ OpenAI API Key 已配置")
	} else {
		log.Println("⚠️  OpenAI API Key 未配置，AI 功能将以降级模式运行")
	}

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
