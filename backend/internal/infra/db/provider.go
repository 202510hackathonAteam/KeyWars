package db

import "keywars/backend/internal/config"

// ExtractDBConfig は、アプリケーション設定からの DB 設定の抽出。
func ExtractDBConfig(cfg *config.Config) *config.DBConfig {
	return &cfg.DB
}
