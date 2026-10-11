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

func TestLoadConfig_DatabaseDefaults(t *testing.T) {
	os.Clearenv()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DBHost != "localhost" {
		t.Errorf("expected localhost, got %s", cfg.DBHost)
	}
	if cfg.DBPort != 5432 {
		t.Errorf("expected 5432, got %d", cfg.DBPort)
	}
	if cfg.DBName != "educore" {
		t.Errorf("expected educore, got %s", cfg.DBName)
	}
	if cfg.DBUser != "postgres" {
		t.Errorf("expected postgres, got %s", cfg.DBUser)
	}
	if cfg.DBSSLMode != "disable" {
		t.Errorf("expected disable, got %s", cfg.DBSSLMode)
	}
	if cfg.DBMaxConns != 10 {
		t.Errorf("expected 10, got %d", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 2 {
		t.Errorf("expected 2, got %d", cfg.DBMinConns)
	}
}

func TestLoadConfig_DatabaseDSN(t *testing.T) {
	os.Clearenv()
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_NAME", "educore_prod")
	os.Setenv("DB_USER", "siakad_user")
	os.Setenv("DB_PASSWORD", "super_secret_password")
	os.Setenv("DB_SSLMODE", "require")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dsn := cfg.DSN()
	if dsn != "postgres://siakad_user:super_secret_password@db.example.com:5433/educore_prod?connect_timeout=5&sslmode=require" {
		t.Errorf("unexpected DSN: %s", dsn)
	}

	sanitized := cfg.SanitizedDSN()
	if sanitized != "postgres://siakad_user:******@db.example.com:5433/educore_prod?connect_timeout=5&sslmode=require" {
		t.Errorf("unexpected SanitizedDSN: %s", sanitized)
	}
}

func TestLoadConfig_InvalidDatabaseConfig(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"InvalidDBPort", map[string]string{"DB_PORT": "99999"}},
		{"InvalidSSLMode", map[string]string{"DB_SSLMODE": "invalid"}},
		{"ZeroMaxConns", map[string]string{"DB_MAX_CONNS": "0"}},
		{"MinConnsGreaterThanMax", map[string]string{"DB_MAX_CONNS": "5", "DB_MIN_CONNS": "10"}},
		{"NegativeConnectTimeout", map[string]string{"DB_CONNECT_TIMEOUT": "-1s"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			_, err := LoadConfig()
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
		})
	}
}
