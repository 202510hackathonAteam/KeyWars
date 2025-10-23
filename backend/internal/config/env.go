package config

import (
	"os"
)

func Load() Config {
	return Config {
		DB: DBConfig{
			User: getEnvOrDefault("MYSQL_USER", "root"),
			Password: getEnvOrDefault("MYSQL_PASSWORD", ""),
			Host: getEnvOrDefault("MYSQL_HOST", "db"),
			Port: getEnvOrDefault("MYSQL_PORT", "3306"),
			Name: getEnvOrDefault("MYSQL_DB", "db"),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}