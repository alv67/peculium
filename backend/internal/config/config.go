package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	RedisAddr     string
	RedisPassword string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	ServerPort int
	ServerHost string

	SeriesMaxPoints int

	LogLevel string

	YahooFinanceEnabled bool
	PriceFetchInterval  time.Duration
	LookupCacheTTL      time.Duration
	ExposureCacheTTL    time.Duration
	YahooMinInterval    time.Duration
	YahooGlobalRate     int
	YahooGlobalWindow   time.Duration

	PythonServiceURL string

	StalePriceDays int
}

func Load() *Config {
	return &Config{
		DBHost:     getEnv("PECULIUM_DB_HOST", "localhost"),
		DBPort:     getEnvInt("PECULIUM_DB_PORT", 5432),
		DBName:     getEnv("PECULIUM_DB_NAME", "peculium"),
		DBUser:     getEnv("PECULIUM_DB_USER", "peculium"),
		DBPassword: getEnvOrFile("PECULIUM_DB_PASSWORD", "peculium"),
		DBSSLMode:  getEnv("PECULIUM_DB_SSLMODE", "disable"),

		RedisAddr:     getEnv("PECULIUM_REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("PECULIUM_REDIS_PASSWORD", ""),

		JWTSecret:     getEnvOrFile("PECULIUM_JWT_SECRET", "change-me-in-production"),
		JWTAccessTTL:  getEnvDuration("PECULIUM_JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL: getEnvDuration("PECULIUM_JWT_REFRESH_TTL", 72*time.Hour),

		ServerPort: getEnvInt("PECULIUM_SERVER_PORT", 8080),
		ServerHost: getEnv("PECULIUM_SERVER_HOST", "0.0.0.0"),

		SeriesMaxPoints: getEnvInt("PECULIUM_SERIES_MAX_POINTS", 500),

		LogLevel: getEnv("PECULIUM_LOG_LEVEL", "debug"),

		YahooFinanceEnabled: getEnvBool("PECULIUM_YAHOO_FINANCE_ENABLED", true),
		PriceFetchInterval:  getEnvDuration("PECULIUM_PRICE_FETCH_INTERVAL", 1*time.Hour),
		LookupCacheTTL:      getEnvDuration("PECULIUM_LOOKUP_CACHE_TTL", 7*24*time.Hour),
		ExposureCacheTTL:    getEnvDuration("PECULIUM_EXPOSURE_CACHE_TTL", 7*24*time.Hour),
		YahooMinInterval:    getEnvDuration("PECULIUM_YAHOO_MIN_INTERVAL", 400*time.Millisecond),
		YahooGlobalRate:     getEnvInt("PECULIUM_YAHOO_GLOBAL_RATE", 8),
		YahooGlobalWindow:   getEnvDuration("PECULIUM_YAHOO_GLOBAL_WINDOW", 1*time.Second),

		PythonServiceURL: getEnv("PECULIUM_PYTHON_SERVICE_URL", "http://python-service:8000"),

		StalePriceDays: getEnvInt("PECULIUM_STALE_PRICE_DAYS", 7),
	}
}

func (c *Config) DSN() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + strconv.Itoa(c.DBPort) + "/" + c.DBName + "?sslmode=" + c.DBSSLMode
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvOrFile resolves a secret value with precedence: explicit env value >
// contents of the file named by <key>_FILE > fallback. The fallback is only
// applied after the file lookup, so a default never masks a *_FILE setting.
func getEnvOrFile(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if v := readSecretFile(os.Getenv(key + "_FILE")); v != "" {
		return v
	}
	return fallback
}

func readSecretFile(path string) string {
	if path == "" {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
