package config

import (
	"os"
	"log"
)

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

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("missing required env: %s", key)
	}
	return value
}