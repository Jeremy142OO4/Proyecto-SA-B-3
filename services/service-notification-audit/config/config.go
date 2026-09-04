package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        string
	DatabaseURL string
	RabbitMQURL string
	JWTSecret   string

	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPAppPassword string
	SMTPFrom        string
}

func LoadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "8085"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/notification_audit_db?sslmode=disable"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		JWTSecret:   getEnv("JWT_SECRET", "super-secret-bank-key-change-in-production"),

		SMTPHost:        getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:        getEnvInt("SMTP_PORT", 587),
		SMTPUsername:    getEnv("SMTP_USERNAME", ""),
		SMTPAppPassword: getEnv("SMTP_APP_PASSWORD", ""),
		SMTPFrom:        getEnv("SMTP_FROM", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	port, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return port
}
