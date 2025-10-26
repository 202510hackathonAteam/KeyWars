package db

import (
	"context"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"keywars/backend/internal/config"
)

type DBConnectionConfig struct {
	MaxOpenConnections int
	MaxIdleConnections int
	ConnectionLifetime time.Duration
	ConnectionIdleTime time.Duration
	LogLevel logger.LogLevel
	PingTimeout time.Duration
}

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

func New(dbSettings config.DBConfig, poolSettings ...DBConnectionConfig) (*gorm.DB, error) {
	connectionSettings := defaultDBConnectionConfig()
	if len(poolSettings) > 0 {
		connectionSettings = poolSettings[0]
	}

	gormDB, err := gorm.Open(mysql.Open(dbSettings.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(connectionSettings.LogLevel),
	})
	if err != nil {
		return nil, err
	}

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

func Close(gormDB *gorm.DB) error {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}