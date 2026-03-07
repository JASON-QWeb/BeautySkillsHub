package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"skill-hub/internal/config"
	"skill-hub/internal/handler"
	"skill-hub/internal/middleware"
	"skill-hub/internal/model"
	"skill-hub/internal/service"
	"skill-hub/internal/service/ai"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect database
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&model.Skill{}, &model.User{}, &model.SkillLike{}, &model.SkillFavorite{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// Backfill legacy rows before human-review fields existed.
	if err := db.Model(&model.Skill{}).
		Where("ai_approved = ? AND (human_review_status IS NULL OR human_review_status = '')", true).
		Updates(map[string]interface{}{
			"human_review_status": model.HumanReviewStatusApproved,
			"published":           true,
		}).Error; err != nil {
		log.Printf("⚠️  迁移旧数据（已通过 AI）失败: %v", err)
	}
	if err := db.Model(&model.Skill{}).
		Where("ai_approved = ? AND (human_review_status IS NULL OR human_review_status = '')", false).
		Updates(map[string]interface{}{
			"human_review_status": model.HumanReviewStatusRejected,
			"published":           false,
		}).Error; err != nil {
		log.Printf("⚠️  迁移旧数据（AI 未通过）失败: %v", err)
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
	mcpHandler := handler.NewResourceHandler(skillSvc, "mcp", cfg)
	toolsHandler := handler.NewResourceHandler(skillSvc, "tools", cfg)
	rulesHandler := handler.NewResourceHandler(skillSvc, "rules", cfg)
	avatarDir := "./avatars"
	authHandler := handler.NewAuthHandler(db, avatarDir)

	// Setup Gin
	r := gin.Default()
	r.Use(middleware.CORS())

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
		publicReads.GET("/skills/:id", skillHandler.GetSkill)
		publicReads.GET("/skills/:id/readme", skillHandler.GetSkillReadme)
		publicReads.GET("/skills/:id/download", skillHandler.DownloadSkill)
		publicReads.POST("/skills/:id/download-hit", skillHandler.TrackDownloadHit)
		// Protected: upload, update & delete require auth
		api.POST("/skills", authHandler.AuthMiddleware(), skillHandler.UploadSkill)
		api.GET("/skills/:id/review-status", authHandler.AuthMiddleware(), skillHandler.GetSkillReviewStatus)
		api.POST("/skills/:id/review/retry", authHandler.AuthMiddleware(), skillHandler.RetrySkillReview)
		api.PUT("/skills/:id", authHandler.AuthMiddleware(), skillHandler.UpdateSkill)
		api.DELETE("/skills/:id", authHandler.AuthMiddleware(), skillHandler.DeleteSkill)
		api.POST("/skills/:id/human-review", authHandler.AuthMiddleware(), skillHandler.HumanReviewSkill)
		api.POST("/skills/:id/like", authHandler.AuthMiddleware(), skillHandler.LikeSkill)
		api.POST("/skills/:id/favorite", authHandler.AuthMiddleware(), skillHandler.AddFavoriteSkill)
		api.DELETE("/skills/:id/favorite", authHandler.AuthMiddleware(), skillHandler.RemoveFavoriteSkill)
		api.GET("/me/favorites", authHandler.AuthMiddleware(), skillHandler.ListMyFavorites)

		// Categories
		api.GET("/categories", skillHandler.GetCategories)

		// Thumbnail serving
		api.GET("/thumbnails/:filename", skillHandler.ServeThumbnail)

		// Avatar serving
		api.GET("/avatars/:filename", authHandler.ServeAvatar)

		// AI Chat
		api.POST("/ai/chat", skillHandler.ChatRecommend)

		// MCP resource routes
		handler.RegisterResourceRoutes(api, mcpHandler, authHandler.AuthMiddleware(), authHandler.OptionalAuthMiddleware(), "/mcps")

		// Tools resource routes
		handler.RegisterResourceRoutes(api, toolsHandler, authHandler.AuthMiddleware(), authHandler.OptionalAuthMiddleware(), "/tools")

		// Rules resource routes
		handler.RegisterResourceRoutes(api, rulesHandler, authHandler.AuthMiddleware(), authHandler.OptionalAuthMiddleware(), "/rules")
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
