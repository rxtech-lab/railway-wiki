//go:build e2e

package e2e

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"

	"github.com/golang-jwt/jwt/v5"
)

// oauthProvider is a minimal in-process OAuth/OIDC identity provider for the
// e2e suite. It generates an RSA keypair, serves an OIDC discovery document and
// the matching JWKS over HTTP so the server-under-test can discover keys from
// the issuer (OAUTH_ISSUER), and mints signed JWT access tokens carrying the
// admin role.
type oauthProvider struct {
	srv    *httptest.Server
	key    *rsa.PrivateKey
	kid    string
	issuer string
}

const e2eKeyID = "e2e-key"

// newOAuthProvider builds the provider and starts its JWKS HTTP server. The
// caller must Close it when done.
func newOAuthProvider() (*oauthProvider, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}
	p := &oauthProvider{key: key, kid: e2eKeyID}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":   p.issuer,
			"jwks_uri": p.jwksURL(),
		})
	})
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p.jwks())
	})
	p.srv = httptest.NewServer(mux)
	p.issuer = p.srv.URL
	return p, nil
}

func (p *oauthProvider) Close() {
	if p.srv != nil {
		p.srv.Close()
	}
}

// jwksURL is the URL the server-under-test fetches signing keys from.
func (p *oauthProvider) jwksURL() string { return p.srv.URL + "/.well-known/jwks.json" }

// jwks renders the public key as a single-key JWK Set.
func (p *oauthProvider) jwks() map[string]any {
	pub := p.key.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	return map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA",
			"use": "sig",
			"alg": "RS256",
			"kid": p.kid,
			"n":   n,
			"e":   e,
		}},
	}
}

// token mints a signed access token for the given subject and role.
func (p *oauthProvider) token(subject, role string) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":   subject,
		"roles": []string{role},
		"iss":   p.issuer,
	})
	tok.Header["kid"] = p.kid
	signed, err := tok.SignedString(p.key)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
