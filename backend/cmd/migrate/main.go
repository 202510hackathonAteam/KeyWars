package main

import (
	"log"

	"keywars/backend/internal/config"
	"keywars/backend/internal/infra/db"
	"keywars/backend/internal/model"
)

// main は、データベースの自動マイグレーションを実行するエントリーポイント。
// internal/model に定義された構造体をもとにテーブルを作成・更新。
func main() {
	// --- 設定ファイルの読み込み ---
	cfg := config.Load()

	// --- DB接続の初期化 ---
	gormDB, err := db.New(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// --- 自動マイグレーションの実行 ---
	if err := gormDB.AutoMigrate(
		// モデル作成後、コメントアウトを解除する
		// &model.User{},
		// &model.Prompt{},
		// 下に他のテーブル構造体を追加していく
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	log.Println("Database migration completed successfully!")
}