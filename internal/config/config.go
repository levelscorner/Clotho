package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	Mode          string        // server, worker, all
	Port          string        // HTTP port
	DatabaseURL   string        // PostgreSQL connection string
	OpenAIKey     string        // OpenAI API key
	GeminiKey     string        // Google AI Studio API key
	OpenRouterKey string        // OpenRouter API key
	OllamaURL     string        // Ollama base URL
	LogLevel      slog.Level    // log level
	JWTSecret     string        // JWT signing secret
	JWTExpiry     time.Duration // JWT access token expiry
	MasterKey     string        // hex-encoded 32-byte envelope encryption master key
}

func Load() (*Config, error) {
	cfg := &Config{
		Mode:          getEnv("MODE", "all"),
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://clotho:clotho@localhost:5432/clotho?sslmode=disable"),
		OpenAIKey:     getEnv("OPENAI_API_KEY", ""),
		GeminiKey:     getEnv("GEMINI_API_KEY", ""),
		OpenRouterKey: getEnv("OPENROUTER_API_KEY", ""),
		OllamaURL:     getEnv("OLLAMA_URL", "http://localhost:11434"),
		LogLevel:      parseLogLevel(getEnv("LOG_LEVEL", "info")),
	}

	cfg.Mode = strings.ToLower(cfg.Mode)
	if cfg.Mode != "server" && cfg.Mode != "worker" && cfg.Mode != "all" {
		return nil, fmt.Errorf("invalid MODE: %s (must be server, worker, or all)", cfg.Mode)
	}

	// JWT
	cfg.JWTSecret = getEnv("JWT_SECRET", "")
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = generateRandomHex(32)
		slog.Warn("JWT_SECRET not set, generated random secret (dev mode only)")
	}
	cfg.JWTExpiry = parseDuration(getEnv("JWT_EXPIRY", "15m"), 15*time.Minute)

	// Envelope encryption master key
	cfg.MasterKey = getEnv("CLOTHO_MASTER_KEY", "")
	if cfg.MasterKey == "" {
		slog.Warn("CLOTHO_MASTER_KEY not set, credentials will be stored without encryption (dev mode only)")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return hex.EncodeToString(b)
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
