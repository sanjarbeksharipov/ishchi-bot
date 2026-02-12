package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Bot      BotConfig
	Database DatabaseConfig
	App      AppConfig
	Payment  PaymentConfig
}

// BotConfig contains Telegram bot specific configuration
type BotConfig struct {
	Token        string
	Verbose      bool
	Poller       time.Duration
	ChannelID    int64
	AdminIDs     []int64
	AdminGroupID int64 // Admin group for payment approvals
	Username     string
	// Webhook configuration
	Mode          string // "webhook" or "polling"
	WebhookURL    string // Public URL for webhook (e.g., https://example.com/webhook)
	WebhookListen string // Listen address for webhook server (e.g., :8443)
	WebhookPort   int    // Port for webhook server
	// Rate limiter configuration
	RateLimitMaxRequests int           // Max requests per window (default: 30)
	RateLimitWindow      time.Duration // Sliding window duration (default: 60s)
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Host           string
	Port           int
	User           string
	Password       string
	DBName         string
	MaxConnections int
}

// AppConfig contains general application configuration
type AppConfig struct {
	Environment string
	LogLevel    string
}

// PaymentConfig contains payment specific configuration
type PaymentConfig struct {
	CardNumber     string
	CardHolderName string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Bot: BotConfig{
			Token:                getEnv("BOT_TOKEN", ""),
			Verbose:              getEnvAsBool("BOT_VERBOSE", false),
			Poller:               getEnvAsDuration("BOT_POLLER", 10*time.Second),
			ChannelID:            getEnvAsInt64("BOT_CHANNEL_ID", 0),
			AdminIDs:             getEnvAsInt64Slice("BOT_ADMIN_IDS", nil),
			AdminGroupID:         getEnvAsInt64("BOT_ADMIN_GROUP_ID", 0),
			Username:             getEnv("BOT_USERNAME", ""),
			Mode:                 getEnv("BOT_MODE", "polling"),
			WebhookURL:           getEnv("BOT_WEBHOOK_URL", ""),
			WebhookListen:        getEnv("BOT_WEBHOOK_LISTEN", ":8443"),
			WebhookPort:          getEnvAsInt("BOT_WEBHOOK_PORT", 8443),
			RateLimitMaxRequests: getEnvAsInt("BOT_RATE_LIMIT_MAX", 30),
			RateLimitWindow:      getEnvAsDuration("BOT_RATE_LIMIT_WINDOW", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:           getEnv("DB_HOST", "localhost"),
			Port:           getEnvAsInt("DB_PORT", 5432),
			User:           getEnv("DB_USER", "postgres"),
			Password:       getEnv("DB_PASSWORD", ""),
			DBName:         getEnv("DB_NAME", "telegram_bot"),
			MaxConnections: getEnvAsInt("DB_MAX_CONNECTIONS", 25),
		},
		App: AppConfig{
			Environment: getEnv("APP_ENV", "development"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
		},
		Payment: PaymentConfig{
			CardNumber:     getEnv("CARD_NUMBER", "8600 0000 0000 0000"),
			CardHolderName: getEnv("CARD_HOLDER_NAME", "ADMIN NAME"),
		},
	}

	if cfg.Bot.Token == "" {
		return nil, fmt.Errorf("BOT_TOKEN environment variable is required")
	}

	return cfg, nil
}

// Helper functions to read environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64Slice(key string, defaultValue []int64) []int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	parts := strings.Split(valueStr, ",")
	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if value, err := strconv.ParseInt(part, 10, 64); err == nil {
			result = append(result, value)
		}
	}
	return result
}

// DSN returns the PostgreSQL connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		d.User, d.Password, d.Host, d.Port, d.DBName)
}
