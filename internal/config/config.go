package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	AppEnv  string
	AppHost string
	AppPort string

	BasicAuthUser string
	BasicAuthPass string

	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string

	StoragePath string

	// Directory where daily JSON-lines log files are written.
	LogPath string

	BroadcastMinDelayMS int
	BroadcastMaxDelayMS int
	BroadcastMaxRetry   int

	// Outbound webhook (WAHA-style). Empty URL disables webhooks.
	WebhookURL      string
	WebhookSecret   string // HMAC-SHA256 signing key (optional)
	WebhookMaxRetry int
	WebhookEvents   string // comma-separated; empty = all

	// Anti-blocking: when true, sends mark-seen + typing presence with a short
	// random delay before delivering a message (mimics human behavior).
	HumanizeSend bool

	// Rate limiting (anti-ban). When enabled, the same recipient can only be
	// messaged once per RateMinIntervalSec; an optional global interval paces
	// all sends. Broadcast uses its own pacing and bypasses the API limiter.
	RateLimitEnabled      bool
	RateMinIntervalSec    int // per-recipient minimum seconds between sends
	RateGlobalIntervalSec int // global minimum seconds between any sends (0 = off)

	// Media retention: auto-delete stored media older than MediaRetentionDays
	// (0 disables). Guards against unbounded disk growth.
	MediaRetentionDays int
}

// Load reads configuration from environment variables, applying sensible
// defaults. A .env file (if present) should be loaded by the caller before
// invoking Load.
func Load() *Config {
	c := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppHost: getEnv("APP_HOST", "0.0.0.0"),
		AppPort: getEnv("APP_PORT", "8080"),

		BasicAuthUser: getEnv("BASIC_AUTH_USERNAME", "admin"),
		BasicAuthPass: getEnv("BASIC_AUTH_PASSWORD", "secret123"),

		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "waproxy"),
		DBPass:    getEnvAllowEmpty("DB_PASSWORD", "waproxy"),
		DBName:    getEnv("DB_NAME", "waproxy"),
		DBSSLMode: getEnv("DB_SSLMODE", "disable"),

		StoragePath: getEnv("STORAGE_PATH", "./storage"),

		LogPath: getEnv("LOG_PATH", "./logs"),

		BroadcastMinDelayMS: getEnvInt("BROADCAST_MIN_DELAY_MS", 3000),
		BroadcastMaxDelayMS: getEnvInt("BROADCAST_MAX_DELAY_MS", 8000),
		BroadcastMaxRetry:   getEnvInt("BROADCAST_MAX_RETRY", 2),

		WebhookURL:      getEnv("WEBHOOK_URL", ""),
		WebhookSecret:   getEnv("WEBHOOK_SECRET", ""),
		WebhookMaxRetry: getEnvInt("WEBHOOK_MAX_RETRY", 3),
		WebhookEvents:   getEnv("WEBHOOK_EVENTS", ""),

		HumanizeSend: getEnvBool("HUMANIZE_SEND", false),

		RateLimitEnabled:      getEnvBool("RATE_LIMIT_ENABLED", true),
		RateMinIntervalSec:    getEnvInt("RATE_MIN_INTERVAL_SEC", 30),
		RateGlobalIntervalSec: getEnvInt("RATE_GLOBAL_INTERVAL_SEC", 0),

		MediaRetentionDays: getEnvInt("MEDIA_RETENTION_DAYS", 14),
	}
	return c
}

// DSN builds a PostgreSQL connection URL for GORM / pgx.
//
// A URL is used instead of the key-value form because the latter mis-parses an
// empty password (e.g. "password= dbname=waproxy" causes pgx to drop dbname and
// fall back to the default database). The URL form is unambiguous.
func (c *Config) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		Host:   c.DBHost + ":" + c.DBPort,
		Path:   "/" + c.DBName,
	}
	// User info: include the password section only when a password is set so an
	// empty password is represented correctly.
	if c.DBPass != "" {
		u.User = url.UserPassword(c.DBUser, c.DBPass)
	} else {
		u.User = url.User(c.DBUser)
	}
	q := url.Values{}
	q.Set("sslmode", c.DBSSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

// AdminDSN is like DSN but targets the default "postgres" maintenance database,
// used to create the application database if it does not exist.
func (c *Config) AdminDSN() string {
	u := url.URL{
		Scheme: "postgres",
		Host:   c.DBHost + ":" + c.DBPort,
		Path:   "/postgres",
	}
	if c.DBPass != "" {
		u.User = url.UserPassword(c.DBUser, c.DBPass)
	} else {
		u.User = url.User(c.DBUser)
	}
	q := url.Values{}
	q.Set("sslmode", c.DBSSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

// Addr returns the host:port the HTTP server should bind to.
func (c *Config) Addr() string {
	return c.AppHost + ":" + c.AppPort
}

// IsProduction reports whether the app is running in production mode.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// getEnvAllowEmpty returns the env value if the key is set (even to an empty
// string), otherwise the fallback. Used for passwords where "" is meaningful
// (e.g. local postgres with trust/no-password auth).
func getEnvAllowEmpty(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}

// WebhookEnabled reports whether outbound webhooks are configured.
func (c *Config) WebhookEnabled() bool {
	return c.WebhookURL != ""
}

// WebhookEventList returns the configured event filter as a slice (empty = all).
func (c *Config) WebhookEventList() []string {
	if strings.TrimSpace(c.WebhookEvents) == "" {
		return nil
	}
	parts := strings.Split(c.WebhookEvents, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
