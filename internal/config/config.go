package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv              string
	AppPort             int
	AppTimezone         string
	LogLevel            string
	HTTPReadTimeout     time.Duration
	HTTPWriteTimeout    time.Duration
	HTTPIdleTimeout     time.Duration
	HTTPShutdownTimeout time.Duration

	// PostgreSQL database configuration
	DBHost            string
	DBPort            int
	DBName            string
	DBUser            string
	DBPassword        string
	DBSSLMode         string
	DBMaxConns        int
	DBMinConns        int
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration
	DBConnectTimeout  time.Duration
	DBHealthTimeout   time.Duration
}

// LoadConfig reads configuration from environment variables and provides safe defaults.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		AppEnv:              getEnvOrDefault("APP_ENV", "development"),
		AppPort:             getEnvAsIntOrDefault("APP_PORT", 8080),
		AppTimezone:         getEnvOrDefault("APP_TIMEZONE", "Asia/Jakarta"),
		LogLevel:            getEnvOrDefault("LOG_LEVEL", "info"),
		HTTPReadTimeout:     getEnvAsDurationOrDefault("HTTP_READ_TIMEOUT", 15*time.Second),
		HTTPWriteTimeout:    getEnvAsDurationOrDefault("HTTP_WRITE_TIMEOUT", 15*time.Second),
		HTTPIdleTimeout:     getEnvAsDurationOrDefault("HTTP_IDLE_TIMEOUT", 60*time.Second),
		HTTPShutdownTimeout: getEnvAsDurationOrDefault("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),

		// Database defaults
		DBHost:            getEnvOrDefault("DB_HOST", "localhost"),
		DBPort:            getEnvAsIntOrDefault("DB_PORT", 5432),
		DBName:            getEnvOrDefault("DB_NAME", "educore"),
		DBUser:            getEnvOrDefault("DB_USER", "postgres"),
		DBPassword:        getEnvOrDefault("DB_PASSWORD", ""),
		DBSSLMode:         getEnvOrDefault("DB_SSLMODE", "disable"),
		DBMaxConns:        getEnvAsIntOrDefault("DB_MAX_CONNS", 10),
		DBMinConns:        getEnvAsIntOrDefault("DB_MIN_CONNS", 2),
		DBMaxConnLifetime: getEnvAsDurationOrDefault("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		DBMaxConnIdleTime: getEnvAsDurationOrDefault("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
		DBConnectTimeout:  getEnvAsDurationOrDefault("DB_CONNECT_TIMEOUT", 5*time.Second),
		DBHealthTimeout:   getEnvAsDurationOrDefault("DB_HEALTH_TIMEOUT", 2*time.Second),
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// DSN returns the standard PostgreSQL connection string formatted as a valid URL.
func (c *Config) DSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   fmt.Sprintf("%s:%d", c.DBHost, c.DBPort),
		Path:   c.DBName,
	}
	q := u.Query()
	q.Set("sslmode", c.DBSSLMode)
	q.Set("connect_timeout", strconv.Itoa(int(c.DBConnectTimeout.Seconds())))
	u.RawQuery = q.Encode()
	return u.String()
}

// SanitizedDSN returns the connection string with password masked for safe logging/display.
func (c *Config) SanitizedDSN() string {
	q := url.Values{}
	q.Set("sslmode", c.DBSSLMode)
	q.Set("connect_timeout", strconv.Itoa(int(c.DBConnectTimeout.Seconds())))
	return fmt.Sprintf("postgres://%s:******@%s:%d/%s?%s", url.QueryEscape(c.DBUser), c.DBHost, c.DBPort, c.DBName, q.Encode())
}

func validate(cfg *Config) error {
	if cfg.AppPort < 1 || cfg.AppPort > 65535 {
		return fmt.Errorf("invalid APP_PORT: %d", cfg.AppPort)
	}
	if cfg.HTTPReadTimeout <= 0 || cfg.HTTPWriteTimeout <= 0 || cfg.HTTPIdleTimeout <= 0 || cfg.HTTPShutdownTimeout <= 0 {
		return fmt.Errorf("HTTP timeouts must be greater than zero")
	}
	if cfg.AppEnv != "development" && cfg.AppEnv != "production" && cfg.AppEnv != "testing" {
		return fmt.Errorf("unsupported APP_ENV: %s", cfg.AppEnv)
	}

	// Database validation
	if cfg.DBHost == "" {
		return fmt.Errorf("DB_HOST cannot be empty")
	}
	if cfg.DBPort < 1 || cfg.DBPort > 65535 {
		return fmt.Errorf("invalid DB_PORT: %d", cfg.DBPort)
	}
	if cfg.DBName == "" {
		return fmt.Errorf("DB_NAME cannot be empty")
	}
	if cfg.DBUser == "" {
		return fmt.Errorf("DB_USER cannot be empty")
	}
	if cfg.DBSSLMode != "disable" && cfg.DBSSLMode != "require" && cfg.DBSSLMode != "verify-ca" && cfg.DBSSLMode != "verify-full" {
		return fmt.Errorf("unsupported DB_SSLMODE: %s", cfg.DBSSLMode)
	}
	if cfg.DBMaxConns <= 0 {
		return fmt.Errorf("DB_MAX_CONNS must be greater than zero")
	}
	if cfg.DBMinConns < 0 || cfg.DBMinConns > cfg.DBMaxConns {
		return fmt.Errorf("DB_MIN_CONNS must be between 0 and DB_MAX_CONNS (%d)", cfg.DBMaxConns)
	}
	if cfg.DBMaxConnLifetime <= 0 || cfg.DBMaxConnIdleTime <= 0 || cfg.DBConnectTimeout <= 0 || cfg.DBHealthTimeout <= 0 {
		return fmt.Errorf("database timeouts and lifetimes must be greater than zero")
	}

	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

func getEnvAsIntOrDefault(key string, fallback int) int {
	strValue := getEnvOrDefault(key, "")
	if strValue == "" {
		return fallback
	}
	value, err := strconv.Atoi(strValue)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvAsDurationOrDefault(key string, fallback time.Duration) time.Duration {
	strValue := getEnvOrDefault(key, "")
	if strValue == "" {
		return fallback
	}
	value, err := time.ParseDuration(strValue)
	if err != nil {
		return fallback
	}
	return value
}
