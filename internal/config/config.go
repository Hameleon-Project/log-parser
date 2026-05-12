package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port   string
	DBConn string
}

func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),
		DBConn: fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "password"),
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_NAME", "logs_db"),
		),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
