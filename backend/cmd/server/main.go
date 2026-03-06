package main

import (
	"fmt"
	"log"

	"skill-hub/internal/config"
	"skill-hub/internal/handler"
	"skill-hub/internal/middleware"
	"skill-hub/internal/model"
	"skill-hub/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
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
	if err := db.AutoMigrate(&model.Skill{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// Initialize services
	skillSvc := service.NewSkillService(db)
	aiSvc := service.NewAIService(cfg)
	githubSyncSvc := service.NewGitHubSyncService(cfg, nil)

	// Initialize handlers
	skillHandler := handler.NewSkillHandler(skillSvc, aiSvc, githubSyncSvc, cfg)

	// Setup Gin
	r := gin.Default()
	r.Use(middleware.CORS())

	// API routes
	api := r.Group("/api")
	{
		// Skill/Resource endpoints
		api.GET("/skills", skillHandler.ListSkills)
		api.GET("/skills/trending", skillHandler.GetTrending)
		api.GET("/skills/:id", skillHandler.GetSkill)
		api.POST("/skills", skillHandler.UploadSkill)
		api.DELETE("/skills/:id", skillHandler.DeleteSkill)
		api.GET("/skills/:id/download", skillHandler.DownloadSkill)

		// Categories
		api.GET("/categories", skillHandler.GetCategories)

		// Thumbnail serving
		api.GET("/thumbnails/:filename", skillHandler.ServeThumbnail)

		// AI Chat
		api.POST("/ai/chat", skillHandler.ChatRecommend)
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
