package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_Default(t *testing.T) {
	os.Clearenv()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("expected development, got %s", cfg.AppEnv)
	}
	if cfg.AppPort != 8080 {
		t.Errorf("expected 8080, got %d", cfg.AppPort)
	}
}

func TestLoadConfig_ValidEnv(t *testing.T) {
	os.Clearenv()
	os.Setenv("APP_PORT", "9090")
	os.Setenv("APP_ENV", "production")
	os.Setenv("HTTP_READ_TIMEOUT", "30s")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.AppPort != 9090 {
		t.Errorf("expected 9090, got %d", cfg.AppPort)
	}
	if cfg.AppEnv != "production" {
		t.Errorf("expected production, got %s", cfg.AppEnv)
	}
	if cfg.HTTPReadTimeout != 30*time.Second {
		t.Errorf("expected 30s, got %v", cfg.HTTPReadTimeout)
	}
}

func TestLoadConfig_InvalidPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("APP_PORT", "70000") // Out of range

	_, err := LoadConfig()
	if err == nil {
		t.Fatalf("expected error for invalid port, got nil")
	}
}

func TestLoadConfig_InvalidTimeout(t *testing.T) {
	os.Clearenv()
	os.Setenv("HTTP_READ_TIMEOUT", "-5s")

	_, err := LoadConfig()
	if err == nil {
		t.Fatalf("expected error for negative timeout, got nil")
	}
}

func TestLoadConfig_InvalidEnv(t *testing.T) {
	os.Clearenv()
	os.Setenv("APP_ENV", "staging") // Not in supported list

	_, err := LoadConfig()
	if err == nil {
		t.Fatalf("expected error for invalid APP_ENV, got nil")
	}
}
