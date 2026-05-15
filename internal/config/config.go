package config

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Port     string
	DBConn   string
	LogLevel slog.Level
}

func Load() *Config {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DB_CONN"))
	}
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/log_db?sslmode=disable"
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	return &Config{
		Port:     port,
		DBConn:   dsn,
		LogLevel: parseLogLevel(os.Getenv("LOG_LEVEL")),
	}
}

func parseLogLevel(v string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
