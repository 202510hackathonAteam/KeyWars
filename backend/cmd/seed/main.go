package main

import (
	"log"

	"keywars/backend/internal/config"
	"keywars/backend/internal/infra/db"
	"keywars/backend/internal/infra/db/initial"
)

// main は、データベースの自動マイグレーションを実行するエントリーポイント。
// internal/domain/entity に定義された構造体をもとにテーブルを作成・更新し、
// internal/db/initial に定義されたマスタデータ（難易度・お題など）を登録します。
func main() {
	// 設定ファイルの読み込み
	cfg := config.Load()

	// DB接続の初期化
	gormDB, err := db.New(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 初期データ投入
	if err := initial.LoadInitialData(gormDB); err != nil {
		log.Fatalf("failed to load initial data: %v", err)
	}

	log.Println("Database migration completed successfully!")
}