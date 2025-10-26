package config

import (
	"os"
	"log"
)

// Loadは、環境変数を読み込み、Config 構造体を生成。
func Load() Config {
	return Config {
		DB: DBConfig{
			User: mustEnv("MYSQL_USER"),
			Password: mustEnv("MYSQL_PASSWORD"),
			Host: mustEnv("MYSQL_HOST"),
			Port: mustEnv("MYSQL_PORT"),
			Name: mustEnv("MYSQL_DATABASE"),
		},
	}
}

// mustEnvは、指定したキーの環境変数を取得。
// 値が存在しない場合はログを出力してアプリケーションを強制終了。
func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("missing required env: %s", key)
	}
	return value
}