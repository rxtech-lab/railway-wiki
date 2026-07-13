package server

import (
	"context"
	"fmt"

	"github.com/rxtech-lab/railway-wiki/internal/auth"
	"github.com/rxtech-lab/railway-wiki/internal/config"
	"github.com/rxtech-lab/railway-wiki/internal/media"
	"github.com/rxtech-lab/railway-wiki/internal/overpass"
)

// ProvideAuthenticator builds the OAuth/OIDC management authenticator. Access
// tokens are verified as JWTs against the identity provider's JWKS.
func ProvideAuthenticator(cfg *config.Config) (auth.Authenticator, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("management auth requires OAUTH_ISSUER")
	}
	return auth.NewJWKSAuthenticator(context.Background(), cfg.Issuer)
}

// ProvideOverpassClient builds the bounded server-side proxy client. Its API
// key is intentionally read only from backend configuration.
func ProvideOverpassClient(cfg *config.Config) (overpass.Client, error) {
	return overpass.NewHTTPClient(overpass.Settings{
		URL:           cfg.OverpassURL,
		APIKey:        cfg.OverpassAPIKey,
		Timeout:       cfg.OverpassTimeout,
		CacheTTL:      cfg.OverpassCacheTTL,
		MaxCandidates: cfg.OverpassMaxCandidates,
		MaxResponse:   cfg.OverpassMaxResponse,
		MinInterval:   cfg.OverpassMinInterval,
		UserAgent:     cfg.OverpassUserAgent,
	})
}

// ProvidePresigner builds the media presigner. When no bucket is configured a
// no-op presigner is returned so the server still starts (uploads then 500).
func ProvidePresigner(cfg *config.Config) (media.Presigner, error) {
	if cfg.S3Bucket == "" {
		return media.NoopPresigner{}, nil
	}
	return media.NewS3Presigner(context.Background(), media.S3Config{
		Bucket:    cfg.S3Bucket,
		Region:    cfg.S3Region,
		Endpoint:  cfg.S3Endpoint,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		PublicURL: cfg.S3PublicURL,
		UsePath:   cfg.S3PathStyle,
		TTL:       cfg.S3TTL,
	})
}
