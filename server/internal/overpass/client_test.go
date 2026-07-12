package overpass

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestSearchBuildsBoundedRailQueryAndCaches(t *testing.T) {
	var calls atomic.Int32
	client, err := NewHTTPClient(Settings{
		URL:           "https://overpass.test/api/interpreter",
		APIKey:        "backend-secret",
		Timeout:       time.Second,
		CacheTTL:      time.Minute,
		MaxCandidates: 5,
		MaxResponse:   4096,
		UserAgent:     "railway-wiki-test",
	})
	require.NoError(t, err)
	client.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		require.Equal(t, "backend-secret", request.Header.Get("X-API-Key"))
		require.Equal(t, "railway-wiki-test", request.Header.Get("User-Agent"))
		body, readErr := io.ReadAll(request.Body)
		require.NoError(t, readErr)
		form, parseErr := url.ParseQuery(string(body))
		require.NoError(t, parseErr)
		query := form.Get("data")
		require.Contains(t, query, `nwr["railway"~"^(station|halt|tram_stop)$"]`)
		require.Contains(t, query, "(40.700000,-74.020000,40.750000,-73.970000)")
		require.Contains(t, query, "out center tags 6;")
		return jsonResponse(`{"elements":[{"type":"way","id":42,"center":{"lat":40.72,"lon":-74.0},"tags":{"railway":"station","name":"Central","name:en":"Central Station","network":"Metro"}}]}`), nil
	})

	request := SearchRequest{Bounds: &Bounds{South: 40.70, West: -74.02, North: 40.75, East: -73.97}, Caller: "admin-a"}
	items, err := client.Search(context.Background(), request)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(42), items[0].ElementID)
	require.Equal(t, "Central", items[0].Name)
	require.Equal(t, "Metro", *items[0].Network)

	items[0].Tags["name"] = "mutated"
	cached, err := client.Search(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "Central", cached[0].Tags["name"])
	require.Equal(t, int32(1), calls.Load())
}

func TestSearchRejectsUnboundedOrOversizedViewport(t *testing.T) {
	client, err := NewHTTPClient(Settings{URL: "https://overpass.test/api/interpreter"})
	require.NoError(t, err)

	_, err = client.Search(context.Background(), SearchRequest{})
	require.ErrorIs(t, err, ErrBadRequest)
	_, err = client.Search(context.Background(), SearchRequest{Bounds: &Bounds{South: 0, West: 0, North: 3, East: 1}})
	require.ErrorIs(t, err, ErrBadRequest)
	_, err = client.Search(context.Background(), SearchRequest{Point: &PointSearch{Latitude: 0, Longitude: 0, RadiusMeters: 10001}})
	require.ErrorIs(t, err, ErrBadRequest)
}

func TestNewHTTPClientRejectsNonHTTPURLs(t *testing.T) {
	for _, value := range []string{"overpass.local/api/interpreter", "/api/interpreter", "ftp://overpass.local/query"} {
		t.Run(value, func(t *testing.T) {
			_, err := NewHTTPClient(Settings{URL: value})
			require.Error(t, err)
		})
	}
}

func TestSearchCoalescesConcurrentRequests(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	client, err := NewHTTPClient(Settings{
		URL: "https://overpass.test/api/interpreter", Timeout: time.Second,
		CacheTTL: time.Minute, MaxResponse: 4096,
	})
	require.NoError(t, err)
	client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return jsonResponse(`{"elements":[{"type":"node","id":1,"lat":1,"lon":2,"tags":{"railway":"station","name":"Central"}}]}`), nil
	})

	request := SearchRequest{Point: &PointSearch{Latitude: 1, Longitude: 2, RadiusMeters: 2000}, Caller: "admin"}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 2)
	wait.Add(1)
	go func() {
		defer wait.Done()
		_, searchErr := client.Search(context.Background(), request)
		errorsSeen <- searchErr
	}()
	<-started
	wait.Add(1)
	go func() {
		defer wait.Done()
		_, searchErr := client.Search(context.Background(), request)
		errorsSeen <- searchErr
	}()
	close(release)
	wait.Wait()
	close(errorsSeen)
	for searchErr := range errorsSeen {
		require.NoError(t, searchErr)
	}
	require.Equal(t, int32(1), calls.Load())
}

func TestSearchThrottlesDistinctUncachedQueriesPerCaller(t *testing.T) {
	client, err := NewHTTPClient(Settings{
		URL: "https://overpass.test/api/interpreter", Timeout: time.Second,
		CacheTTL: time.Minute, MaxResponse: 4096, MinInterval: time.Minute,
	})
	require.NoError(t, err)
	client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(`{"elements":[]}`), nil
	})

	_, err = client.Search(context.Background(), SearchRequest{
		Point: &PointSearch{Latitude: 1, Longitude: 2, RadiusMeters: 2000}, Caller: "admin",
	})
	require.NoError(t, err)
	_, err = client.Search(context.Background(), SearchRequest{
		Point: &PointSearch{Latitude: 2, Longitude: 3, RadiusMeters: 2000}, Caller: "admin",
	})
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestExecuteClassifiesTimeoutAndUpstreamFailures(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		client, err := NewHTTPClient(Settings{
			URL: "https://overpass.test/api/interpreter", Timeout: 5 * time.Millisecond, MaxResponse: 4096,
		})
		require.NoError(t, err)
		client.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
			<-request.Context().Done()
			return nil, request.Context().Err()
		})
		_, err = client.Fetch(context.Background(), "node", 1)
		require.ErrorIs(t, err, ErrTimeout)
	})

	t.Run("status", func(t *testing.T) {
		client, err := NewHTTPClient(Settings{URL: "https://overpass.test/api/interpreter", MaxResponse: 4096})
		require.NoError(t, err)
		client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader("unavailable")),
				Header:     make(http.Header),
			}, nil
		})
		_, err = client.Fetch(context.Background(), "node", 1)
		require.ErrorIs(t, err, ErrUnavailable)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		client, err := NewHTTPClient(Settings{URL: "https://overpass.test/api/interpreter", MaxResponse: 4096})
		require.NoError(t, err)
		client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(`{"elements":`), nil
		})
		_, err = client.Fetch(context.Background(), "node", 1)
		require.ErrorIs(t, err, ErrUnavailable)
	})
}

func TestNormalizeExcludesEntrancesAndBusOnlyFeatures(t *testing.T) {
	latitude, longitude := 1.0, 2.0
	for _, tags := range []map[string]string{
		{"railway": "subway_entrance", "station": "subway", "name": "Entrance A"},
		{"station": "subway", "entrance": "yes", "name": "Entrance B"},
		{"amenity": "bus_station", "bus": "yes", "name": "Bus only"},
		{"amenity": "ferry_terminal", "ferry": "yes", "name": "Ferry only"},
	} {
		_, ok := normalizeElement(overpassElement{Type: "node", ID: 1, Lat: &latitude, Lon: &longitude, Tags: tags})
		require.False(t, ok, "unexpected candidate for tags %#v", tags)
	}

	candidate, ok := normalizeElement(overpassElement{
		Type: "node", ID: 2, Lat: &latitude, Lon: &longitude,
		Tags: map[string]string{"railway": "tram_stop", "name": "Tram"},
	})
	require.True(t, ok)
	require.Equal(t, "Tram", candidate.Name)
}

func TestFetchRefusesNonRailwayElement(t *testing.T) {
	client, err := NewHTTPClient(Settings{URL: "https://overpass.test/api/interpreter", MaxResponse: 4096})
	require.NoError(t, err)
	client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(`{"elements":[{"type":"node","id":7,"lat":1,"lon":2,"tags":{"amenity":"cafe","name":"Not a station"}}]}`), nil
	})
	_, err = client.Fetch(context.Background(), "node", 7)
	require.ErrorIs(t, err, ErrBadRequest)
}

func TestExecuteCapsResponseBody(t *testing.T) {
	client, err := NewHTTPClient(Settings{URL: "https://overpass.test/api/interpreter", MaxResponse: 16})
	require.NoError(t, err)
	client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 17))), Header: make(http.Header)}, nil
	})
	_, err = client.Fetch(context.Background(), "node", 1)
	require.True(t, errors.Is(err, ErrTooLarge))
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}
