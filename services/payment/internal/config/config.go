package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	Host        string
	DatabaseURL string
	NATSUrl     string
	StripeKey   string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file, reading from environment")
	}

	dbURL, err := mustEnv("DATABASE_URL")

	if err != nil {
		return Config{}, err
	}

	natsURL, err := mustEnv("NATS_URL")
	if err != nil {
		return Config{}, err
	}

	stripeKey, err := mustEnv("STRIPE_KEY")
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:        getEnv("PORT", "6005"),
		Host:        getEnv("HOST", "localhost"),
		DatabaseURL: dbURL,
		NATSUrl:     natsURL,
		StripeKey:   stripeKey,
	}, nil
}

func mustEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env var missing", "key", key)
		return "", &missingEnvError{key}
	}
	return v, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type missingEnvError struct{ key string }

func (e *missingEnvError) Error() string {
	return "required env var missing: " + e.key
}
