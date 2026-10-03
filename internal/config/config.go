package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                string
	HTTPAddr           string
	DatabaseURL        string
	MigrateDatabaseURL string
	DemoPassword       string
	AIProvider         string
	GeminiAPIKey       string
	GeminiModel        string
	AITimeout          time.Duration
	AIMaxInputChars    int
	AIMaxOutputTokens  int
	AIDailyCallCap     int
	CORSOrigin         string
	CookieSecure       bool
	WorkerEmbedded     bool
	BodyLimit          int64
	RateLimitPerMin    int
	LoginLimitPerMin   int
	AssistantRetention time.Duration
	SessionTTL         time.Duration
	TrustProxy         bool
}

func Load() (Config, error) {
	cfg := Config{
		Env:                getenv("ENV", "development"),
		HTTPAddr:           getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		MigrateDatabaseURL: os.Getenv("MIGRATE_DATABASE_URL"),
		DemoPassword:       getenv("DEMO_PASSWORD", "synthetic-demo-pass"),
		AIProvider:         getenv("AI_PROVIDER", "auto"),
		GeminiAPIKey:       os.Getenv("GEMINI_API_KEY"),
		GeminiModel:        getenv("GEMINI_MODEL", "gemini-2.5-flash"),
		AITimeout:          20 * time.Second,
		AIMaxInputChars:    24_000,
		AIMaxOutputTokens:  1500,
		AIDailyCallCap:     200,
		CORSOrigin:         os.Getenv("CORS_ORIGIN"),
		CookieSecure:       os.Getenv("COOKIE_SECURE") == "true",
		WorkerEmbedded:     getenv("WORKER_EMBEDDED", "true") != "false",
		BodyLimit:          1_000_000,
		RateLimitPerMin:    120,
		LoginLimitPerMin:   20,
		AssistantRetention: 30 * 24 * time.Hour,
		SessionTTL:         12 * time.Hour,
		TrustProxy:         os.Getenv("TRUST_PROXY") == "true",
	}
	if v := os.Getenv("RATE_LIMIT_PER_MIN"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("RATE_LIMIT_PER_MIN must be a positive integer")
		}
		cfg.RateLimitPerMin = n
	}
	if v := os.Getenv("AI_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("AI_TIMEOUT: %w", err)
		}
		cfg.AITimeout = d
	}
	if v := os.Getenv("AI_MAX_INPUT_CHARS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("AI_MAX_INPUT_CHARS: %w", err)
		}
		cfg.AIMaxInputChars = n
	}
	if v := os.Getenv("AI_DAILY_CALL_CAP"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("AI_DAILY_CALL_CAP: %w", err)
		}
		cfg.AIDailyCallCap = n
	}
	if v := os.Getenv("BODY_LIMIT_BYTES"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("BODY_LIMIT_BYTES: %w", err)
		}
		cfg.BodyLimit = n
	}
	if cfg.MigrateDatabaseURL == "" {
		cfg.MigrateDatabaseURL = cfg.DatabaseURL
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	switch c.AIProvider {
	case "auto", "stub", "gemini":
	default:
		return fmt.Errorf("AI_PROVIDER must be auto, stub, or gemini")
	}
	if c.Env == "production" {
		if strings.Contains(c.DatabaseURL, "finguard_app:finguard_app") {
			return fmt.Errorf("refusing default local database credentials when ENV=production")
		}
		if c.DemoPassword == "synthetic-demo-pass" {
			return fmt.Errorf("refusing the default demo password when ENV=production")
		}
		if !c.CookieSecure {
			return fmt.Errorf("COOKIE_SECURE must be true when ENV=production")
		}
	}
	return nil
}

func (c Config) LiveModel() bool {
	if c.AIProvider == "stub" {
		return false
	}
	if c.AIProvider == "gemini" {
		return c.GeminiAPIKey != ""
	}
	return c.GeminiAPIKey != ""
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
