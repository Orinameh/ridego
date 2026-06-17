package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string // HTTP + WebSocket port
	GRPCPort      string // gRPC port for Matching Service
	Host          string
	RedisAddr     string
	RedisPassword string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file, reading from environment")
	}

	return Config{
		Port:          getEnv("PORT", "6003"),
		GRPCPort:      getEnv("GRPC_PORT", "50051"),
		Host:          getEnv("HOST", "localhost"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
