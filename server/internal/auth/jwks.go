package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// JWKSAuthenticator validates signed JWT access tokens against the identity
// provider's JWKS and extracts the caller's role from the `role` claim. The
// JWKS endpoint is discovered from the issuer via OIDC discovery, and the
// issuer is enforced on every token.
type JWKSAuthenticator struct {
	kf     keyfunc.Keyfunc
	issuer string
}

// NewJWKSAuthenticator resolves the issuer's JWKS endpoint via OIDC discovery
// (issuer + /.well-known/openid-configuration) and loads its signing keys.
func NewJWKSAuthenticator(ctx context.Context, issuer string) (*JWKSAuthenticator, error) {
	if issuer == "" {
		return nil, fmt.Errorf("issuer is required")
	}
	jwksURL, err := discoverJWKSURL(ctx, issuer)
	if err != nil {
		return nil, err
	}
	kf, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to load JWKS from %s: %w", jwksURL, err)
	}
	return &JWKSAuthenticator{kf: kf, issuer: issuer}, nil
}

// discoverJWKSURL fetches the issuer's OIDC discovery document and returns its
// jwks_uri.
func discoverJWKSURL(ctx context.Context, issuer string) (string, error) {
	configURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configURL, nil)
	if err != nil {
		return "", fmt.Errorf("build OIDC discovery request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch OIDC discovery from %s: %w", configURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC discovery %s returned status %d", configURL, resp.StatusCode)
	}
	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("decode OIDC discovery from %s: %w", configURL, err)
	}
	if doc.JWKSURI == "" {
		return "", fmt.Errorf("OIDC discovery %s has no jwks_uri", configURL)
	}
	return doc.JWKSURI, nil
}

// Authenticate parses and verifies the token, returning the caller principal.
func (a *JWKSAuthenticator) Authenticate(_ context.Context, token string) (*Principal, error) {
	parsed, err := jwt.Parse(token, a.kf.Keyfunc, jwt.WithIssuer(a.issuer))
	if err != nil || !parsed.Valid {
		return nil, ErrUnauthorized
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrUnauthorized
	}
	sub, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	return &Principal{Subject: sub, Role: role}, nil
}
