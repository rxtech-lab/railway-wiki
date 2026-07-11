// Package config loads application configuration from the environment.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration.
type Config struct {
	// Server
	Port string

	// Database — a libSQL/Turso URL (libsql://host?authToken=…) or a local
	// SQLite DSN (file:railway.db, :memory:).
	DatabaseURL string

	// Management auth (OAuth/OIDC). Management endpoints require a JWT access
	// token verified against the issuer's JWKS, discovered via OIDC discovery.
	Issuer string // token issuer (OAUTH_ISSUER), required

	// Media object storage (S3-compatible). Empty S3Bucket disables uploads.
	S3Bucket    string
	S3Region    string
	S3Endpoint  string // optional; set for localstack / S3-compatible stores
	S3AccessKey string
	S3SecretKey string
	S3PublicURL string
	S3PathStyle bool
	S3TTL       time.Duration
}

// Load reads configuration from environment variables, applying defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "file:railway.db"),

		Issuer: getEnv("OAUTH_ISSUER", ""),

		S3Bucket:    getEnv("S3_BUCKET", ""),
		S3Region:    getEnv("S3_REGION", "us-east-1"),
		S3Endpoint:  getEnv("S3_ENDPOINT", ""),
		S3AccessKey: getEnv("S3_ACCESS_KEY_ID", ""),
		S3SecretKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
		S3PublicURL: getEnv("S3_PUBLIC_URL", ""),
		S3PathStyle: getBool("S3_PATH_STYLE", false),
		S3TTL:       getDuration("S3_PRESIGN_TTL", 15*time.Minute),
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
