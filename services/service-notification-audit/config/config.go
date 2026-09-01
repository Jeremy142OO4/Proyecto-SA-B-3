package config

import (
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	RabbitMQURL string
	JWTSecret   string
}

func LoadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "8085"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/notification_audit_db?sslmode=disable"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		JWTSecret:   getEnv("JWT_SECRET", "super-secret-bank-key-change-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
