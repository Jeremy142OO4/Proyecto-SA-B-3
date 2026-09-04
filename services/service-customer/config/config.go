package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	DatabaseURL        string
	RabbitMQURL        string
	JWTSecret          string
	JWTExpirationHours int
	ActivationLinkBase string
}

func LoadConfig() *Config {
	jwtExp, err := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))
	if err != nil {
		jwtExp = 24
	}

	return &Config{
		Port:               getEnv("PORT", "8081"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/customer_db?sslmode=disable"),
		RabbitMQURL:        getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		JWTSecret:          getEnv("JWT_SECRET", "super-secret-bank-key-change-in-production"),
		JWTExpirationHours: jwtExp,
		ActivationLinkBase: getEnv("ACTIVATION_LINK_BASE", "http://localhost:5173/activate"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
