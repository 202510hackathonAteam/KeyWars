package redisx

import (
	"keywars/backend/internal/config"
)

// ExtractRedisConfig は、アプリケーション設定からの Redis 設定の抽出。
func ExtractRedisConfig(cfg *config.Config) *config.RedisConfig {
	return &cfg.Redis
}
