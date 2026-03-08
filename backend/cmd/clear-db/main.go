package main

import (
	"fmt"

	"skill-hub/internal/config"
	"skill-hub/internal/logging"
	"skill-hub/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type purgeStats struct {
	Users          int64
	Skills         int64
	SkillLikes     int64
	SkillFavorites int64
}

func main() {
	cfg := config.Load()
	logging.Init(cfg.AppEnv)

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logging.Fatal("数据库连接失败", "error", err)
	}

	stats, err := purgeAllData(db)
	if err != nil {
		logging.Fatal("清理失败", "error", err)
	}

	fmt.Printf("已清空数据库数据（DB: %s）\n", cfg.DatabaseURL)
	fmt.Printf("- users: %d\n", stats.Users)
	fmt.Printf("- skills: %d\n", stats.Skills)
	fmt.Printf("- skill_likes: %d\n", stats.SkillLikes)
	fmt.Printf("- skill_favorites: %d\n", stats.SkillFavorites)
}

func purgeAllData(db *gorm.DB) (purgeStats, error) {
	stats := purgeStats{}

	tx := db.Begin()
	if tx.Error != nil {
		return stats, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Model(&model.SkillFavorite{}).Count(&stats.SkillFavorites).Error; err != nil {
		tx.Rollback()
		return stats, err
	}
	if err := tx.Model(&model.SkillLike{}).Count(&stats.SkillLikes).Error; err != nil {
		tx.Rollback()
		return stats, err
	}
	if err := tx.Model(&model.Skill{}).Count(&stats.Skills).Error; err != nil {
		tx.Rollback()
		return stats, err
	}
	if err := tx.Model(&model.User{}).Count(&stats.Users).Error; err != nil {
		tx.Rollback()
		return stats, err
	}

	del := tx.Session(&gorm.Session{AllowGlobalUpdate: true})
	if err := del.Delete(&model.SkillFavorite{}).Error; err != nil {
		tx.Rollback()
		return stats, err
	}
	if err := del.Delete(&model.SkillLike{}).Error; err != nil {
		tx.Rollback()
		return stats, err
	}
	if err := del.Delete(&model.Skill{}).Error; err != nil {
		tx.Rollback()
		return stats, err
	}
	if err := del.Delete(&model.User{}).Error; err != nil {
		tx.Rollback()
		return stats, err
	}

	if err := tx.Exec(
		"TRUNCATE TABLE skill_favorites, skill_likes, skills, users RESTART IDENTITY CASCADE",
	).Error; err != nil {
		tx.Rollback()
		return stats, err
	}

	if err := tx.Commit().Error; err != nil {
		return stats, err
	}

	return stats, nil
}
