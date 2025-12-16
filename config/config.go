package config

import (
	"os"
	"strings"
)

type Config struct {
	Port            string
	Environment     string
	AllowedOrigins  []string
	JWTSecret       string
	Redis           RedisConfig
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func Load() *Config {
	// Parse allowed origins (comma-separated)
	originsStr := getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	origins := strings.Split(originsStr, ",")

	return &Config{
		Port:           getEnv("PORT", "8080"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		AllowedOrigins: origins,
		JWTSecret:      getEnv("JWT_SECRET", "change-me-in-production"),
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
