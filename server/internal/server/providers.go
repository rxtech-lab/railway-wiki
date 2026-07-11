package server

import (
	"context"
	"fmt"

	"github.com/rxtech-lab/railway-wiki/internal/auth"
	"github.com/rxtech-lab/railway-wiki/internal/config"
	"github.com/rxtech-lab/railway-wiki/internal/media"
)

// ProvideAuthenticator builds the OAuth/OIDC management authenticator. Access
// tokens are verified as JWTs against the identity provider's JWKS.
func ProvideAuthenticator(cfg *config.Config) (auth.Authenticator, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("management auth requires OAUTH_ISSUER")
	}
	return auth.NewJWKSAuthenticator(context.Background(), cfg.Issuer)
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
