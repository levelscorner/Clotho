package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	OllamaURL      string        // Ollama base URL
	KokoroURL      string        // Kokoro-FastAPI base URL (local TTS)
	ComfyUIURL     string        // ComfyUI base URL (local image gen)
	ReplicateToken string        // Replicate API token
	LogLevel       slog.Level    // log level
	JWTSecret     string        // JWT signing secret
	JWTExpiry     time.Duration // JWT access token expiry
	MasterKey     string        // hex-encoded 32-byte envelope encryption master key
	AdminPassword string        // admin user password (from ADMIN_PASSWORD env var)

	// DataDir is the root directory for on-disk pipeline outputs; honours
	// CLOTHO_DATA_DIR. Defaults to "$HOME/Documents/Clotho" on all platforms.
	// When the home directory cannot be resolved, falls back to "./clotho-data".
	DataDir string

	// NoAuth enables local-dev authentication bypass when true.
	// Requires AcknowledgeNoAuth to also be true (fail-closed).
	NoAuth            bool
	AcknowledgeNoAuth bool
}

// prodMarkers are environment variables whose presence indicates a
// production-like deployment. If any are set while NoAuth=true, Validate
// returns an error.
var prodMarkers = []string{
	"KUBERNETES_SERVICE_HOST",
	"RAILWAY_ENVIRONMENT",
	"RENDER_SERVICE_ID",
	"VERCEL",
}

// Validate performs cross-field checks that cannot be expressed by Load.
// In particular, it ensures NoAuth cannot accidentally be enabled in a
// production environment.
func (c *Config) Validate() error {
	if c.DataDir == "" {
		return fmt.Errorf("DataDir must be set (CLOTHO_DATA_DIR or default)")
	}

	if !c.NoAuth {
		return nil
	}

	if !c.AcknowledgeNoAuth {
		return fmt.Errorf("NO_AUTH=true requires CLOTHO_ACKNOWLEDGE_NO_AUTH=true to explicitly acknowledge unauthenticated mode")
	}

	if strings.EqualFold(os.Getenv("NODE_ENV"), "production") {
		return fmt.Errorf("NO_AUTH=true is forbidden when NODE_ENV=production")
	}

	for _, key := range prodMarkers {
		if os.Getenv(key) != "" {
			return fmt.Errorf("NO_AUTH=true is forbidden when %s is set", key)
		}
	}

	return nil
}

func Load() (*Config, error) {
	cfg := &Config{
		Mode:          getEnv("MODE", "all"),
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://clotho:clotho@localhost:5432/clotho?sslmode=disable"),
		OpenAIKey:     getEnv("OPENAI_API_KEY", ""),
		GeminiKey:     getEnv("GEMINI_API_KEY", ""),
		OpenRouterKey: getEnv("OPENROUTER_API_KEY", ""),
		OllamaURL:      getEnv("OLLAMA_URL", "http://localhost:11434"),
		KokoroURL:      getEnv("KOKORO_URL", "http://localhost:8880"),
		ComfyUIURL:     getEnv("COMFYUI_URL", "http://localhost:8188"),
		ReplicateToken: getEnv("REPLICATE_API_TOKEN", ""),
		LogLevel:       parseLogLevel(getEnv("LOG_LEVEL", "info")),
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

	// Admin password
	cfg.AdminPassword = getEnv("ADMIN_PASSWORD", "clotho123")

	// Data directory for on-disk pipeline outputs.
	defaultDataDir, err := defaultUserDataDir()
	if err != nil || defaultDataDir == "" {
		// Fallback to a working-directory subdir if HOME can't be resolved.
		defaultDataDir = filepath.Join(".", "clotho-data")
		slog.Warn("could not resolve user home directory; falling back to ./clotho-data for CLOTHO_DATA_DIR")
	}
	cfg.DataDir = getEnv("CLOTHO_DATA_DIR", defaultDataDir)

	// Auth bypass (local-dev only; fail-closed via Validate)
	cfg.NoAuth = isTruthy(getEnv("NO_AUTH", ""))
	cfg.AcknowledgeNoAuth = isTruthy(getEnv("CLOTHO_ACKNOWLEDGE_NO_AUTH", ""))

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.NoAuth {
		slog.Warn("UNAUTHENTICATED MODE ENABLED — do not use with real data (NO_AUTH=true)")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// isTruthy parses common truthy string representations case-insensitively.
// Returns true for: "true", "yes", "y", "on", "1". False for everything
// else, including the empty string. Used for NO_AUTH-style opt-in flags
// where users reasonably type any of the above forms.
func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "yes", "y", "on", "1":
		return true
	default:
		return false
	}
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

// defaultUserDataDir returns "$HOME/Documents/Clotho" on all platforms. When
// the home directory cannot be resolved it returns "" and a non-nil error so
// the caller can pick a working-directory fallback.
func defaultUserDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", fmt.Errorf("user home directory is empty")
	}
	return filepath.Join(home, "Documents", "Clotho"), nil
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
