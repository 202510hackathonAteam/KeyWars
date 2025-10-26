package config

import (
	"fmt"
)

// DBConfig は、データベース接続に必要な設定情報の定義
type DBConfig struct {
	User string
	Password string
	Host string
	Port string
	Name string
}

// ServerConfig は、アプリケーションサーバーに関する設定の定義
type ServerConfig struct {
	Port string
}

// Config は、アプリ全体の設定をまとめた構造体。
type Config struct {
	DB DBConfig
	Server ServerConfig
}

// DSN は、MySQL 用の接続文字列（Data Source Name）の生成
func (db DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		db.User, db.Password, db.Host, db.Port, db.Name,
	)
}