package config

import (
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	DatabaseDSN string
	JWTSecret   string
	Environment string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		DatabaseDSN: getEnv("DATABASE_DSN", "school.db"),
		JWTSecret:   getEnv("JWT_SECRET", "school-desk-secret-key-2024"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
