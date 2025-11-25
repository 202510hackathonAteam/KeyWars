package config

import (
	"fmt"
	"os"
	"time"
)

// DBConfig は、データベース接続に必要な設定情報の定義
type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// JWTConfig は、JWT の発行および検証に必要な設定値の定義。
type JWTConfig struct {
	IssuerName         string
	HMACSecretKey      []byte
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// LoadJWTConfig は、JWT 認証ハンドラの初期化
func LoadJWTConfig() JWTConfig {
	return JWTConfig{
		IssuerName:         "keywars",
		HMACSecretKey:      []byte(os.Getenv("JWT_SECRET")),
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 30 * 24 * time.Hour,
	}
}

// CookieConfig は、アプリケーションで使用する Cookie の共通設定を保持する構造体。
type CookieConfig struct {
	Domain string
	SameSite string
	Secure bool
}

// LoadCookieConfig は、CookieConfig のデフォルト設定を読み込む初期化関数。
func LoadCookieConfig() CookieConfig {
	return CookieConfig{
		// 本番環境では必ずhttp.SameSiteStrictModeにすること
		SameSite: http.SameSiteNoneMode,
		// 本番環境ではドメイン名を記載すること
		Domain: os.Getenv("COOKIE_DOMAIN"),
		// 本番環境では必ずtrueにすること
		// Secure: false,
		Secure: os.Getenv("COOKIE_SECURE") == "true",
	}
}

// Config は、アプリ全体の設定をまとめた構造体。
type Config struct {
	DB    DBConfig
	Redis RedisConfig
	JWT   JWTConfig
}

// DSN は、MySQL 用の接続文字列（Data Source Name）の生成
func (db DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		db.User, db.Password, db.Host, db.Port, db.Name,
	)
}
