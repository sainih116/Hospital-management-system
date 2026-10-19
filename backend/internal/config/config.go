// Package config centralizes application configuration loaded from environment
// variables (with an optional .env file for local development).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the HMS backend.
type Config struct {
	Env             string
	Port            string
	AppBaseURL      string
	FrontendBaseURL string
	AllowedOrigins  []string
	LogLevel        string
	Mongo           MongoConfig
	Redis           RedisConfig
	JWT             JWTConfig
	ShutdownTimeout time.Duration
}

// MongoConfig holds MongoDB connection settings.
type MongoConfig struct {
	URI            string
	Database       string
	ConnectTimeout time.Duration
}

// RedisConfig holds Redis connection settings. Redis is optional; when
// Enabled is false the application must run without a cache layer.
type RedisConfig struct {
	Enabled  bool
	Addr     string
	Password string
	DB       int
}

// JWTConfig holds secrets and TTLs for access/refresh token issuance.
type JWTConfig struct {
	AccessSecret    string
	RefreshSecret   string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
}

// Load reads configuration from the environment, optionally hydrating it
// from a .env file first (missing .env files are not an error).
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Env:             getEnv("APP_ENV", "development"),
		Port:            getEnv("PORT", "5000"),
		AppBaseURL:      getEnv("APP_BASE_URL", "http://localhost:5000"),
		FrontendBaseURL: getEnv("FRONTEND_BASE_URL", "http://localhost:5173"),
		AllowedOrigins:  splitAndTrim(getEnv("ALLOWED_ORIGINS", "http://localhost:5173")),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		Mongo: MongoConfig{
			URI:            getEnv("MONGO_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGO_DATABASE", "hms"),
			ConnectTimeout: getDurationSeconds("MONGO_CONNECT_TIMEOUT_SECONDS", 10),
		},
		Redis: RedisConfig{
			Enabled:  getBool("REDIS_ENABLED", false),
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			AccessSecret:    getEnv("JWT_ACCESS_SECRET", ""),
			RefreshSecret:   getEnv("JWT_REFRESH_SECRET", ""),
			AccessTokenTTL:  getDurationMinutes("JWT_ACCESS_TTL_MINUTES", 15),
			RefreshTokenTTL: getDurationMinutes("JWT_REFRESH_TTL_MINUTES", 60*24*7),
			Issuer:          getEnv("JWT_ISSUER", "hms-backend"),
		},
		ShutdownTimeout: getDurationSeconds("SHUTDOWN_TIMEOUT_SECONDS", 15),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate enforces invariants the application must never start without.
// JWT secrets are checked unconditionally (not just in production): an
// empty HMAC secret lets anyone forge a valid access token, and that risk
// doesn't depend on APP_ENV being set correctly — a misconfigured or
// forgotten APP_ENV must never silently downgrade this to a warning.
func (c *Config) validate() error {
	if c.JWT.AccessSecret == "" || c.JWT.RefreshSecret == "" {
		return fmt.Errorf("config: JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must be set — refusing to start with an empty JWT signing secret (see .env.example)")
	}
	return nil
}

// IsProduction reports whether the app is running with APP_ENV=production.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Env, "production")
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDurationSeconds(key string, fallbackSeconds int) time.Duration {
	return time.Duration(getInt(key, fallbackSeconds)) * time.Second
}

func getDurationMinutes(key string, fallbackMinutes int) time.Duration {
	return time.Duration(getInt(key, fallbackMinutes)) * time.Minute
}

func splitAndTrim(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
