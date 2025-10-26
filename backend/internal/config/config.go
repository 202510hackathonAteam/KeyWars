package config

import (
	"fmt"
)

type DBConfig struct {
	User string
	Password string
	Host string
	Port string
	Name string
}

type ServerConfig struct {
	Port string
}

type Config struct {
	DB DBConfig
	Server ServerConfig
}

func (db DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		db.User, db.Password, db.Host, db.Port, db.Name,
	)
}