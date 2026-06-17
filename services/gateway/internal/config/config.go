package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Port         string
	Host         string
	RedisAddr    string
	JWTPublicKey *rsa.PublicKey
}

func Load() (Config, error) {
	pubPEMRaw, err := mustEnv("JWT_PUBLIC_KEY")
	if err != nil {
		return Config{}, err
	}
	// Replace literal "\n" text from .env with real newlines
	pubPEM := strings.ReplaceAll(pubPEMRaw, `\n`, "\n")

	pubBlock, _ := pem.Decode([]byte(pubPEM))
	if pubBlock == nil {
		return Config{}, fmt.Errorf("decode public key: invalid or poorly formatted PEM block data")
	}

	parsedPubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return Config{}, fmt.Errorf("parse public key structure: %w", err)
	}

	pubKey, ok := parsedPubKey.(*rsa.PublicKey)
	if !ok {
		return Config{}, fmt.Errorf("public key structure validation: key is not a valid RSA type")
	}

	return Config{
		Port:         getEnv("PORT", "8080"),
		Host:         getEnv("HOST", "localhost"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		JWTPublicKey: pubKey,
	}, nil
}

func mustEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env var missing", "key", key)
		return "", fmt.Errorf("required env var missing: %s", key)
	}
	return v, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
