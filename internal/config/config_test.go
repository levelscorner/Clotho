package config

import (
	"log/slog"
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Unset env vars that might interfere
	envVars := []string{"MODE", "PORT", "DATABASE_URL", "OPENAI_API_KEY", "GEMINI_API_KEY", "OPENROUTER_API_KEY", "OLLAMA_URL", "LOG_LEVEL"}
	for _, key := range envVars {
		orig, exists := os.LookupEnv(key)
		if exists {
			os.Unsetenv(key)
			defer os.Setenv(key, orig)
		}
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Mode != "all" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "all")
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.DatabaseURL != "postgres://clotho:clotho@localhost:5432/clotho?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want default", cfg.DatabaseURL)
	}
	if cfg.OpenAIKey != "" {
		t.Errorf("OpenAIKey = %q, want empty", cfg.OpenAIKey)
	}
	if cfg.GeminiKey != "" {
		t.Errorf("GeminiKey = %q, want empty", cfg.GeminiKey)
	}
	if cfg.OpenRouterKey != "" {
		t.Errorf("OpenRouterKey = %q, want empty", cfg.OpenRouterKey)
	}
	if cfg.OllamaURL != "http://localhost:11434" {
		t.Errorf("OllamaURL = %q, want %q", cfg.OllamaURL, "http://localhost:11434")
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
}

func TestLoad_InvalidMode(t *testing.T) {
	orig, exists := os.LookupEnv("MODE")
	os.Setenv("MODE", "invalid")
	if exists {
		defer os.Setenv("MODE", orig)
	} else {
		defer os.Unsetenv("MODE")
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid MODE, got nil")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	envs := map[string]string{
		"MODE":               "server",
		"PORT":               "9090",
		"DATABASE_URL":       "postgres://custom:custom@db:5432/custom",
		"OPENAI_API_KEY":     "sk-test-123",
		"GEMINI_API_KEY":     "gem-test-456",
		"OPENROUTER_API_KEY": "or-test-789",
		"OLLAMA_URL":         "http://gpu-server:11434",
		"LOG_LEVEL":          "debug",
	}

	for key, val := range envs {
		orig, exists := os.LookupEnv(key)
		os.Setenv(key, val)
		if exists {
			defer os.Setenv(key, orig)
		} else {
			defer os.Unsetenv(key)
		}
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Mode != "server" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "server")
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.DatabaseURL != "postgres://custom:custom@db:5432/custom" {
		t.Errorf("DatabaseURL = %q, want custom", cfg.DatabaseURL)
	}
	if cfg.OpenAIKey != "sk-test-123" {
		t.Errorf("OpenAIKey = %q, want %q", cfg.OpenAIKey, "sk-test-123")
	}
	if cfg.GeminiKey != "gem-test-456" {
		t.Errorf("GeminiKey = %q, want %q", cfg.GeminiKey, "gem-test-456")
	}
	if cfg.OpenRouterKey != "or-test-789" {
		t.Errorf("OpenRouterKey = %q, want %q", cfg.OpenRouterKey, "or-test-789")
	}
	if cfg.OllamaURL != "http://gpu-server:11434" {
		t.Errorf("OllamaURL = %q, want %q", cfg.OllamaURL, "http://gpu-server:11434")
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
}

func TestLoad_ValidModes(t *testing.T) {
	validModes := []string{"server", "worker", "all", "SERVER", "Worker", "ALL"}

	for _, mode := range validModes {
		orig, exists := os.LookupEnv("MODE")
		os.Setenv("MODE", mode)

		cfg, err := Load()
		if err != nil {
			t.Errorf("MODE=%q: unexpected error: %v", mode, err)
		} else {
			// All modes should be lowercased
			if cfg.Mode != "server" && cfg.Mode != "worker" && cfg.Mode != "all" {
				t.Errorf("MODE=%q: got cfg.Mode=%q, want lowercased valid mode", mode, cfg.Mode)
			}
		}

		if exists {
			os.Setenv("MODE", orig)
		} else {
			os.Unsetenv("MODE")
		}
	}
}
