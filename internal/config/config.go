package config

import (
	"fmt"
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
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
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
