package main

import (
	"log"

	"keywars/backend/internal/config"
	"keywars/backend/internal/infra/db"
	"keywars/backend/internal/infra/db/initial"
	"keywars/backend/internal/infra/db/seed"
)

// main は、初期データなどをデータベースへ自動登録を実行するエントリーポイント。
// internal/infra/db/initial 配下に定義されたマスタデータ（例: 難易度・お題など）を
// データベースに登録します。
func main() {
	// 設定ファイルの読み込み
	cfg := config.Load()

	// DB接続の初期化
	gormDB, err := db.New(&cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 初期データ投入
	if err := initial.LoadInitialData(gormDB); err != nil {
		log.Fatalf("failed to load initial data: %v", err)
	}

	// テストデータ投入（本番環境では削除）
	if err := seed.SeedDevelopmentData(gormDB); err != nil {
		log.Fatalf("failed to load demo data: %v", err)
	}

	log.Println("Database migration completed successfully!")
}