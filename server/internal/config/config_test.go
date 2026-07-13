package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadSkipRoleCheck(t *testing.T) {
	t.Setenv("SKIP_ROLE_CHECK", "true")
	t.Setenv("E2E_MODE", "false")

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.SkipRoleCheck)
	require.False(t, cfg.E2EMode)
}

func TestLoadOverpassDefaults(t *testing.T) {
	for _, key := range []string{
		"OVERPASS_UPSTREAM_URL", "OVERPASS_API_KEY", "OVERPASS_TIMEOUT",
		"OVERPASS_CACHE_TTL", "OVERPASS_MAX_CANDIDATES",
		"OVERPASS_MAX_RESPONSE_BYTES", "OVERPASS_MIN_INTERVAL",
		"OVERPASS_USER_AGENT",
	} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "https://overpass-api.de/api/interpreter", cfg.OverpassURL)
	require.Empty(t, cfg.OverpassAPIKey)
	require.Equal(t, 20*time.Second, cfg.OverpassTimeout)
	require.Equal(t, 5*time.Minute, cfg.OverpassCacheTTL)
	require.Equal(t, 500, cfg.OverpassMaxCandidates)
	require.Equal(t, int64(5*1024*1024), cfg.OverpassMaxResponse)
	require.Equal(t, time.Second, cfg.OverpassMinInterval)
	require.NotEmpty(t, cfg.OverpassUserAgent)
}

func TestLoadOverpassOverrides(t *testing.T) {
	t.Setenv("OVERPASS_UPSTREAM_URL", "https://maps.example.test/overpass/api/interpreter")
	t.Setenv("OVERPASS_API_KEY", "backend-key")
	t.Setenv("OVERPASS_TIMEOUT", "12s")
	t.Setenv("OVERPASS_CACHE_TTL", "2m")
	t.Setenv("OVERPASS_MAX_CANDIDATES", "123")
	t.Setenv("OVERPASS_MAX_RESPONSE_BYTES", "2048")
	t.Setenv("OVERPASS_MIN_INTERVAL", "3s")
	t.Setenv("OVERPASS_USER_AGENT", "railway-wiki-test")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "https://maps.example.test/overpass/api/interpreter", cfg.OverpassURL)
	require.Equal(t, "backend-key", cfg.OverpassAPIKey)
	require.Equal(t, 12*time.Second, cfg.OverpassTimeout)
	require.Equal(t, 2*time.Minute, cfg.OverpassCacheTTL)
	require.Equal(t, 123, cfg.OverpassMaxCandidates)
	require.Equal(t, int64(2048), cfg.OverpassMaxResponse)
	require.Equal(t, 3*time.Second, cfg.OverpassMinInterval)
	require.Equal(t, "railway-wiki-test", cfg.OverpassUserAgent)
}
