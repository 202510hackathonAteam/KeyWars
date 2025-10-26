package db

import (
	"context"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"keywars/backend/internal/config"
)

// DBConnectionConfig は、データベース接続プールおよびログ出力に関する設定値の定義。
type DBConnectionConfig struct {
	MaxOpenConnections int
	MaxIdleConnections int
	ConnectionLifetime time.Duration
	ConnectionIdleTime time.Duration
	LogLevel logger.LogLevel
	PingTimeout time.Duration
}

// defaultDBConnectionConfig は、接続プールおよびログ設定のデフォルト値を生成。
func defaultDBConnectionConfig() DBConnectionConfig {
	return DBConnectionConfig{
		MaxOpenConnections: 25,
		MaxIdleConnections: 25,
		ConnectionLifetime: 55 * time.Minute,
		ConnectionIdleTime: 15 * time.Minute,
		LogLevel: logger.Warn,
		PingTimeout: 5 * time.Second,
	}
}

// New は、MySQL への接続を初期化し、*gorm.DB を生成。
// オプションで接続プール設定を上書き可能。
// 接続確認 (Ping) に失敗した場合はエラーを返却。
func New(dbSettings config.DBConfig, poolSettings ...DBConnectionConfig) (*gorm.DB, error) {
	// --- 接続プール設定の読み込み（引数指定がなければデフォルト値を使用）---
	connectionSettings := defaultDBConnectionConfig()
	if len(poolSettings) > 0 {
		connectionSettings = poolSettings[0]
	}

	// --- 接続プール設定の読み込み（引数指定がなければデフォルト値を使用）---
	gormDB, err := gorm.Open(mysql.Open(dbSettings.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(connectionSettings.LogLevel),
	})
	if err != nil {
		return nil, err
	}

	// --- SQL DB の取得とプール設定 ---
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(connectionSettings.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(connectionSettings.MaxIdleConnections)
	if connectionSettings.ConnectionLifetime > 0 {
		sqlDB.SetConnMaxLifetime(connectionSettings.ConnectionLifetime)
	}
	if connectionSettings.ConnectionIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(connectionSettings.ConnectionIdleTime)
	}

	// --- 接続確認処理（最大 60 秒間リトライ）---
  deadline := time.Now().Add(60 * time.Second)
  for {
    ctx, cancel := context.WithTimeout(context.Background(), connectionSettings.PingTimeout)
    err := sqlDB.PingContext(ctx)
    cancel()
    if err == nil {
      break
    }
    if time.Now().After(deadline) {
      return nil, err
    }
    time.Sleep(1 * time.Second)
  }


	return gormDB, nil
}

// Close は、GORM 経由で開かれたデータベース接続を安全にクローズ。
// *gorm.DB から *sql.DB を取得し、Close() を呼び出す。
func Close(gormDB *gorm.DB) error {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}