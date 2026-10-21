package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	Blnk        BlnkConfig
	Chapa       ChapaConfig
	CORS        CORSConfig
	Platform    PlatformConfig
	Server      ServerConfig
	SMS         SMSConfig
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

type BlnkConfig struct {
	BaseURL          string
	SecretKey        string
	ETBLedgerID      string
	EscrowBalanceID  string
	RevenueBalanceID string

	ProcessingFeeBalanceID string
}

type ChapaConfig struct {
	Mode          string
	BaseURL       string
	SecretKey     string
	WebhookSecret string
	CallbackURL   string
	ReturnURL     string
}

const (
	ChapaModeLive = "live"
	ChapaModeMock = "mock"
)

type PlatformConfig struct {
	FeePercent               int64
	PaymentTimeout           time.Duration
	AutoConfirmDeliveryDays  int
	PayoutMinCents           int64
	PayoutDailyLimitCents    int64
	RequirePhoneVerification bool
	Timezone                 *time.Location
}

const (
	SMSProviderLog         = "log"
	SMSProviderAfroMessage = "afromessage"
)

type SMSConfig struct {
	Provider          string
	AfroMessageURL    string
	AfroMessageToken  string
	AfroMessageSender string
	AfroMessageFrom   string
	OTPDevCode        string
	OTPSecret         string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type ServerConfig struct {
	TrustedProxyCount int
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
		Blnk: BlnkConfig{
			BaseURL:          getEnv("BLNK_BASE_URL", "http://localhost:5001"),
			SecretKey:        os.Getenv("BLNK_SECRET_KEY"),
			ETBLedgerID:      os.Getenv("BLNK_ETB_LEDGER_ID"),
			EscrowBalanceID:  os.Getenv("BLNK_ESCROW_BALANCE_ID"),
			RevenueBalanceID: os.Getenv("BLNK_REVENUE_BALANCE_ID"),

			ProcessingFeeBalanceID: os.Getenv("BLNK_PROCESSING_FEE_BALANCE_ID"),
		},
		Chapa: ChapaConfig{
			Mode:          getEnv("CHAPA_MODE", ChapaModeLive),
			BaseURL:       getEnv("CHAPA_BASE_URL", "https://api.chapa.co/v1"),
			SecretKey:     os.Getenv("CHAPA_SECRET_KEY"),
			WebhookSecret: os.Getenv("CHAPA_WEBHOOK_SECRET"),
			CallbackURL:   getEnv("CHAPA_CALLBACK_URL", ""),
			ReturnURL:     getEnv("CHAPA_RETURN_URL", ""),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseStringSlice(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"), []string{"http://localhost:3000", "http://localhost:5173"}),
		},
		Server: ServerConfig{
			TrustedProxyCount: parseInt(getEnv("TRUSTED_PROXY_COUNT", "0"), 0),
		},
		Platform: PlatformConfig{
			FeePercent:               int64(parseInt(getEnv("PLATFORM_FEE_PERCENT", "5"), 5)),
			PaymentTimeout:           parseDuration(getEnv("ORDER_PAYMENT_TIMEOUT", "1h"), time.Hour),
			AutoConfirmDeliveryDays:  parseInt(getEnv("AUTO_CONFIRM_DELIVERY_DAYS", "7"), 7),
			PayoutMinCents:           int64(parseInt(getEnv("PAYOUT_MIN_CENTS", "10000"), 10000)),
			PayoutDailyLimitCents:    int64(parseInt(getEnv("PAYOUT_DAILY_LIMIT_CENTS", "10000000"), 10000000)),
			RequirePhoneVerification: getEnv("REQUIRE_PHONE_VERIFICATION", "true") == "true",
		},
		SMS: SMSConfig{
			Provider:          getEnv("SMS_PROVIDER", SMSProviderLog),
			AfroMessageURL:    getEnv("AFROMESSAGE_BASE_URL", "https://api.afromessage.com/api"),
			AfroMessageToken:  os.Getenv("AFROMESSAGE_TOKEN"),
			AfroMessageSender: os.Getenv("AFROMESSAGE_SENDER"),
			AfroMessageFrom:   os.Getenv("AFROMESSAGE_IDENTIFIER_ID"),
			OTPDevCode:        os.Getenv("OTP_DEV_CODE"),
			OTPSecret:         getEnv("OTP_SECRET", "dev-otp-secret"),
		},
	}

	tz, err := time.LoadLocation(getEnv("APP_TIMEZONE", "Africa/Addis_Ababa"))
	if err != nil {
		return nil, fmt.Errorf("invalid APP_TIMEZONE: %w", err)
	}
	cfg.Platform.Timezone = tz

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
	if c.Platform.FeePercent < 0 || c.Platform.FeePercent > 100 {
		return fmt.Errorf("PLATFORM_FEE_PERCENT must be between 0 and 100")
	}
	switch c.Chapa.Mode {
	case ChapaModeLive, ChapaModeMock:
	default:
		return fmt.Errorf("CHAPA_MODE must be %q or %q", ChapaModeLive, ChapaModeMock)
	}
	if c.Chapa.Mode == ChapaModeMock && !c.IsDevelopment() {
		return fmt.Errorf("CHAPA_MODE=mock is only allowed when ENVIRONMENT=development")
	}
	if err := c.validateMarketplace(); err != nil {
		return err
	}
	if c.Chapa.WebhookSecret == "" && !c.IsDevelopment() {
		return fmt.Errorf("CHAPA_WEBHOOK_SECRET environment variable is required")
	}
	if !c.IsDevelopment() {
		return c.validateProduction()
	}
	return nil
}

func (c *Config) validateMarketplace() error {
	if c.Platform.AutoConfirmDeliveryDays < 1 {
		return fmt.Errorf("AUTO_CONFIRM_DELIVERY_DAYS must be at least 1")
	}
	if c.Platform.PayoutMinCents < 100 {
		return fmt.Errorf("PAYOUT_MIN_CENTS must be at least 100 (1 ETB)")
	}
	switch c.SMS.Provider {
	case SMSProviderLog, SMSProviderAfroMessage:
	default:
		return fmt.Errorf("SMS_PROVIDER must be %q or %q", SMSProviderLog, SMSProviderAfroMessage)
	}
	if c.SMS.Provider == SMSProviderAfroMessage && c.SMS.AfroMessageToken == "" {
		return fmt.Errorf("AFROMESSAGE_TOKEN is required when SMS_PROVIDER=afromessage")
	}
	return nil
}

// validateProduction refuses development shortcuts and requires every secret
// and ledger balance a live deployment needs.
func (c *Config) validateProduction() error {
	if c.SMS.OTPDevCode != "" {
		return fmt.Errorf("OTP_DEV_CODE is only allowed when ENVIRONMENT=development")
	}
	if c.SMS.Provider == SMSProviderLog {
		return fmt.Errorf("SMS_PROVIDER=log is only allowed when ENVIRONMENT=development")
	}
	if c.SMS.OTPSecret == "dev-otp-secret" || len(c.SMS.OTPSecret) < 32 {
		return fmt.Errorf("OTP_SECRET must be set to a random value of at least 32 characters")
	}

	required := map[string]string{
		"CHAPA_SECRET_KEY":               c.Chapa.SecretKey,
		"BLNK_ETB_LEDGER_ID":             c.Blnk.ETBLedgerID,
		"BLNK_ESCROW_BALANCE_ID":         c.Blnk.EscrowBalanceID,
		"BLNK_REVENUE_BALANCE_ID":        c.Blnk.RevenueBalanceID,
		"BLNK_PROCESSING_FEE_BALANCE_ID": c.Blnk.ProcessingFeeBalanceID,
	}
	for name, value := range required {
		if value == "" {
			return fmt.Errorf("%s environment variable is required", name)
		}
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

func parseStringSlice(s string, defaultValue []string) []string {
	if s == "" {
		return defaultValue
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}
