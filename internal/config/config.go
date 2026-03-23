package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	Environment string
	DatabaseURL string
	RedisURL    string
	JWT         JWTConfig
	Argon2      Argon2Config
	RateLimit   RateLimitConfig
	Logger      LoggerConfig
}

type JWTConfig struct {
	AccessPrivateKeyPath  string
	AccessPublicKeyPath   string
	RefreshPrivateKeyPath string
	RefreshPublicKeyPath  string
	AccessTTL             time.Duration
	RefreshTTL            time.Duration
}

type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

type LoggerConfig struct {
	Format string
	Level  string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWT: JWTConfig{
			AccessPrivateKeyPath:  getEnv("JWT_ACCESS_PRIVATE_KEY_PATH", "keys/access_private.pem"),
			AccessPublicKeyPath:   getEnv("JWT_ACCESS_PUBLIC_KEY_PATH", "keys/access_public.pem"),
			RefreshPrivateKeyPath: getEnv("JWT_REFRESH_PRIVATE_KEY_PATH", "keys/refresh_private.pem"),
			RefreshPublicKeyPath:  getEnv("JWT_REFRESH_PUBLIC_KEY_PATH", "keys/refresh_public.pem"),
			AccessTTL:             parseDuration(getEnv("JWT_ACCESS_TTL", "15m"), 15*time.Minute),
			RefreshTTL:            parseDuration(getEnv("JWT_REFRESH_TTL", "168h"), 168*time.Hour),
		},
		Argon2: Argon2Config{
			Memory:      parseUint32(getEnv("ARGON2_MEMORY", "65536"), 65536),
			Iterations:  parseUint32(getEnv("ARGON2_ITERATIONS", "3"), 3),
			Parallelism: parseUint8(getEnv("ARGON2_PARALLELISM", "2"), 2),
			SaltLength:  parseUint32(getEnv("ARGON2_SALT_LENGTH", "16"), 16),
			KeyLength:   parseUint32(getEnv("ARGON2_KEY_LENGTH", "32"), 32),
		},
		RateLimit: RateLimitConfig{
			Requests: parseInt(getEnv("RATE_LIMIT_REQUESTS", "100"), 100),
			Window:   parseDuration(getEnv("RATE_LIMIT_WINDOW", "1m"), time.Minute),
		},
		Logger: LoggerConfig{
			Format: getEnv("LOG_FORMAT", "text"),
			Level:  getEnv("LOG_LEVEL", "info"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL environment variable is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string, defaultValue int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultValue
}

func parseUint32(s string, defaultValue uint32) uint32 {
	if v, err := strconv.ParseUint(s, 10, 32); err == nil {
		return uint32(v)
	}
	return defaultValue
}

func parseUint8(s string, defaultValue uint8) uint8 {
	if v, err := strconv.ParseUint(s, 10, 8); err == nil {
		return uint8(v)
	}
	return defaultValue
}

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return defaultValue
}
