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

	// Midtrans Snap.
	// ServerKey dari dashboard Midtrans (Settings > Access Keys). Sandbox dan
	// produksi memakai host berbeda, jadi base URL-nya eksplisit.
	MidtransServerKey string
	// MidtransAuthBaseURL dipakai halaman Snap: app.sandbox.midtrans.com
	// (sandbox) atau app.midtrans.com (produksi).
	MidtransAuthBaseURL string
	// MidtransAPIBaseURL dipakai Get Status API: api.sandbox.midtrans.com
	// (sandbox) atau api.midtrans.com (produksi). Kosong = diturunkan dari
	// auth base URL dengan mengganti host app. -> api.
	MidtransAPIBaseURL string

	// JWKS (security.md bagian 2). JWKSURL kosong = <SupabaseURL>/auth/v1/.well-known/jwks.json.
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
		AppPort:             get("APP_PORT", "8080"),
		AppEnv:              get("APP_ENV", "development"),
		JEVBaseURL:          get("JEV_BASE_URL", "https://api.experientiallabs.ai/v1/systemone"),
		BaseURL:             get("BASE_URL", ""),
		DatabaseURL:         get("DATABASE_URL", ""),
		SupabaseURL:         get("SUPABASE_URL", ""),
		SupabaseServiceKey:  get("SUPABASE_SERVICE_ROLE_KEY", ""),
		JEVAPIKey:           get("JEV_API_KEY", ""),
		MidtransServerKey:   get("MIDTRANS_SERVER_KEY", ""),
		MidtransAuthBaseURL: get("MIDTRANS_AUTH_BASE_URL", "https://app.sandbox.midtrans.com"),
		MidtransAPIBaseURL:  get("MIDTRANS_API_BASE_URL", ""),
		JWKSURL:             get("SUPABASE_JWKS_URL", ""),
	}

	if c.MidtransServerKey == "" {
		return nil, errors.New("MIDTRANS_SERVER_KEY wajib diisi (dashboard Midtrans > Settings > Access Keys)")
	}
	// Host Snap tidak boleh tertukar antara sandbox dan produksi: uang nyata
	// bergantung padanya. Terima hanya dua nilai yang dikenal.
	switch c.MidtransAuthBaseURL {
	case "https://app.sandbox.midtrans.com", "https://app.midtrans.com":
	case "https://app.midtrans.com/", "https://app.sandbox.midtrans.com/":
		c.MidtransAuthBaseURL = strings.TrimRight(c.MidtransAuthBaseURL, "/")
	default:
		return nil, fmt.Errorf("MIDTRANS_AUTH_BASE_URL harus app.sandbox.midtrans.com atau app.midtrans.com: %q", c.MidtransAuthBaseURL)
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
