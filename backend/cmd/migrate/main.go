package main

import (
	"flag"
	"log"

	"skill-hub/internal/config"
	"skill-hub/internal/database"
)

func main() {
	cfg := config.Load()

	databaseURL := flag.String("database-url", cfg.DatabaseURL, "PostgreSQL connection string")
	migrationsDir := flag.String("migrations-dir", "../db/migrations", "Directory containing SQL migrations")
	flag.Parse()

	migrator, err := database.NewMigrator(*databaseURL, *migrationsDir)
	if err != nil {
		log.Fatalf("初始化迁移器失败: %v", err)
	}

	if err := migrator.Up(); err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}

	log.Printf("数据库已迁移到最新版本（source=%s）", migrator.SourceURL())
}
