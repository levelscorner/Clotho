package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Mode          string     // server, worker, all
	Port          string     // HTTP port
	DatabaseURL   string     // PostgreSQL connection string
	OpenAIKey     string     // OpenAI API key
	GeminiKey     string     // Google AI Studio API key
	OpenRouterKey string     // OpenRouter API key
	OllamaURL     string     // Ollama base URL
	LogLevel      slog.Level // log level
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

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
