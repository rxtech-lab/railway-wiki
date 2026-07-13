package k8s

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayUsesIndependentRouteKeys(t *testing.T) {
	gateway := readFixture(t, "gateway.yaml")
	secret := readFixture(t, "secrets.example.yaml")

	mapsLocation := nginxBlock(t, gateway, "location /maps/")
	overpassLocation := nginxBlock(t, gateway, "location /overpass/")
	healthLocation := nginxBlock(t, gateway, "location = /healthz")
	defaultLocation := nginxBlock(t, gateway, "location /")
	mapsKeyMap := nginxBlock(t, secret, "map $http_x_api_key $maps_api_key_valid")
	overpassKeyMap := nginxBlock(t, secret, "map $http_x_api_key $overpass_api_key_valid")

	require.Contains(t, mapsLocation, "$maps_api_key_valid")
	require.NotContains(t, mapsLocation, "$overpass_api_key_valid")
	require.Contains(t, mapsKeyMap, "REPLACE_WITH_IOS_MAPS_KEY")
	require.NotContains(t, mapsKeyMap, "REPLACE_WITH_BACKEND_KEY")

	require.Contains(t, overpassLocation, "$overpass_api_key_valid")
	require.NotContains(t, overpassLocation, "$maps_api_key_valid")
	require.Contains(t, overpassKeyMap, "REPLACE_WITH_BACKEND_KEY")
	require.NotContains(t, overpassKeyMap, "REPLACE_WITH_IOS_MAPS_KEY")

	require.Contains(t, healthLocation, "return 200")
	require.NotContains(t, healthLocation, "api_key_valid")
	require.Contains(t, defaultLocation, "return 404")
}

func TestGatewayForwardsAuthenticatedHeaderAndKeepsMapResourcesScoped(t *testing.T) {
	gateway := readFixture(t, "gateway.yaml")
	tileserver := readFixture(t, "tileserver.yaml")

	for _, marker := range []string{"location /maps/", "location /overpass/"} {
		location := nginxBlock(t, gateway, marker)
		require.Contains(t, location, "proxy_set_header X-API-Key $http_x_api_key;")
	}
	require.Contains(t, tileserver, "--public_url")
	require.Contains(t, tileserver, "$(TILESERVER_PUBLIC_URL)")
	require.Contains(t, tileserver, "https://maps.example.com/maps/")
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	require.NoError(t, err)
	return string(data)
}

func nginxBlock(t *testing.T, source, marker string) string {
	t.Helper()
	markerIndex := strings.Index(source, marker)
	require.NotEqual(t, -1, markerIndex, "missing nginx block %q", marker)
	openOffset := strings.IndexByte(source[markerIndex:], '{')
	require.NotEqual(t, -1, openOffset, "missing opening brace for %q", marker)
	start := markerIndex + openOffset
	depth := 0
	for index := start; index < len(source); index++ {
		switch source[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[markerIndex : index+1]
			}
		}
	}
	t.Fatalf("unterminated nginx block %q", marker)
	return ""
}
