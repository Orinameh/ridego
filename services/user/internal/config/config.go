package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	Host          string
	DatabaseURL   string
	JWTPrivateKey *rsa.PrivateKey
	JWTPublicKey  *rsa.PublicKey
}

func Load() (Config, error) {
	// Load .env file — silently ignored in production
	// where env vars are injected by Kubernetes
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file, reading from environment")
	}
	privPEMRaw, err := mustEnv("JWT_PRIVATE_KEY")
	if err != nil {
		return Config{}, err
	}
	// Safely replace literal "\n" string text from .env with real system newlines
	privPEM := strings.ReplaceAll(privPEMRaw, `\n`, "\n")

	pubPEMRaw, err := mustEnv("JWT_PUBLIC_KEY")
	if err != nil {
		return Config{}, err
	}
	// Safely replace literal "\n" string text from .env with real system newlines
	pubPEM := strings.ReplaceAll(pubPEMRaw, `\n`, "\n")

	dbURL, err := mustEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	// --- Decode Private Key ---
	privBlock, _ := pem.Decode([]byte(privPEM))
	if privBlock == nil {
		return Config{}, fmt.Errorf("decode private key: invalid or poorly formatted PEM block data")
	}

	parsedPrivKey, err := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
	if err != nil {
		return Config{}, fmt.Errorf("parse private key structure: %w", err)
	}

	privKey, ok := parsedPrivKey.(*rsa.PrivateKey)
	if !ok {
		return Config{}, fmt.Errorf("private key structure validation: key is not a valid RSA type")
	}

	// --- Decode Public Key ---
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
		Port:          getEnv("PORT", "6001"),
		Host:          getEnv("HOST", "localhost"),
		DatabaseURL:   dbURL,
		JWTPrivateKey: privKey,
		JWTPublicKey:  pubKey,
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
