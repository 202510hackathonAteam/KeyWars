package main

import (
	"log"

	"keywars/backend/internal/config"
	"keywars/backend/internal/infra/db"
	"keywars/backend/internal/model"
)

func main() {
	cfg := config.Load()

	gormDB, err := db.New(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// --- 自動マイグレーションを実行 ---
	if err := gormDB.AutoMigrate(
		// モデル作成後、コメントアウトを解除する
		// &model.User{},
		// &model.Prompt{},
		// ここに他のテーブル構造体を追加していく
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	log.Println("Database migration completed successfully!")
}