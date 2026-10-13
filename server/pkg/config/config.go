package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config adalah satu struct konfigurasi. Nilai dibaca dari environment (.env lokal
// dimuat lebih dulu bila ada). Tidak ada default untuk secret: kosong = gagal start.
type Config struct {
	AppPort string
	AppEnv  string
	BaseURL string

	DatabaseURL string

	SupabaseURL        string
	SupabaseServiceKey string

	JEVAPIKey string
	// JEVBaseURL override gateway JEV; default https://api.experientiallabs.ai/v1/systemone
	JEVBaseURL string

	DokuClientID   string
	DokuSecretKey  string
	DokuBaseURL    string
	DokuNotifyPath string
	DokuIsProd     bool

	// JWKS (security.md bagian 2). JWKSURL kosong = <SupabaseURL>/.well-known/jwks.json.
	JWKSURL string
}

// Load membaca .env lalu environment. path .env yang hilang bukan error.
func Load(envFile string) (*Config, error) {
	if envFile != "" {
		if err := loadEnvFile(envFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	c := &Config{
		AppPort:            get("APP_PORT", "8080"),
		AppEnv:             get("APP_ENV", "development"),
		JEVBaseURL:         get("JEV_BASE_URL", "https://api.experientiallabs.ai/v1/systemone"),
		BaseURL:            get("BASE_URL", ""),
		DatabaseURL:        get("DATABASE_URL", ""),
		SupabaseURL:        get("SUPABASE_URL", ""),
		SupabaseServiceKey: get("SUPABASE_SERVICE_ROLE_KEY", ""),
		JEVAPIKey:          get("JEV_API_KEY", ""),
		DokuClientID:       get("DOKU_CLIENT_ID", ""),
		DokuSecretKey:      get("DOKU_SECRET_KEY", ""),
		DokuBaseURL:        get("DOKU_BASE_URL", "https://api-sandbox.doku.com"),
		DokuNotifyPath:     get("DOKU_NOTIFICATION_PATH", "/v1/webhooks/doku"),
		DokuIsProd:         get("DOKU_ENVIRONMENT", "sandbox") == "production",
		JWKSURL:            get("SUPABASE_JWKS_URL", ""),
	}

	if c.DokuNotifyPath == "" || !strings.HasPrefix(c.DokuNotifyPath, "/") {
		return nil, fmt.Errorf("DOKU_NOTIFICATION_PATH harus path absolut: %q", c.DokuNotifyPath)
	}
	if _, err := strconv.Atoi(c.AppPort); err != nil {
		return nil, fmt.Errorf("APP_PORT bukan angka: %q", c.AppPort)
	}
	return c, nil
}

func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

func get(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// loadEnvFile memuat pasangan KEY=VALUE sederhana. Baris "#" dan baris kosong diabaikan.
// Nilai environment yang sudah ada tidak ditimpa. ponytail: parser minimal, ganti ke
// viper bila butuh quoting/escape/var-expansion.
func loadEnvFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
	return nil
}
