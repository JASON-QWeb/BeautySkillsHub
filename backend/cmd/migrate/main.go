package main

import (
	"flag"
	"log/slog"

	"skill-hub/internal/config"
	"skill-hub/internal/database"
	"skill-hub/internal/logging"
)

func main() {
	cfg := config.Load()
	logging.Init(cfg.AppEnv)

	databaseURL := flag.String("database-url", cfg.DatabaseURL, "PostgreSQL connection string")
	migrationsDir := flag.String("migrations-dir", "../db/migrations", "Directory containing SQL migrations")
	flag.Parse()

	migrator, err := database.NewMigrator(*databaseURL, *migrationsDir)
	if err != nil {
		logging.Fatal("初始化迁移器失败", "error", err)
	}

	if err := migrator.Up(); err != nil {
		logging.Fatal("执行迁移失败", "error", err)
	}

	slog.Info("数据库已迁移到最新版本", "source", migrator.SourceURL())
}
