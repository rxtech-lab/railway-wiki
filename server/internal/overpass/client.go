package overpass

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	items     []Candidate
	expiresAt time.Time
}

type inFlight struct {
	done  chan struct{}
	items []Candidate
	err   error
}

// HTTPClient adds response caps, caching, request coalescing, and per-caller
// throttling around the upstream interpreter endpoint.
type HTTPClient struct {
	settings Settings
	client   *http.Client
	mu       sync.Mutex
	cache    map[string]cacheEntry
	flight   map[string]*inFlight
	last     map[string]time.Time
}

func NewHTTPClient(settings Settings) (*HTTPClient, error) {
	parsedURL, err := url.ParseRequestURI(settings.URL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		if err == nil {
			err = errors.New("URL must be an absolute HTTP(S) URL")
		}
		return nil, fmt.Errorf("invalid overpass URL: %w", err)
	}
	if settings.Timeout <= 0 {
		settings.Timeout = 20 * time.Second
	}
	if settings.CacheTTL <= 0 {
		settings.CacheTTL = 5 * time.Minute
	}
	if settings.MaxCandidates <= 0 {
		settings.MaxCandidates = 500
	}
	if settings.MaxResponse <= 0 {
		settings.MaxResponse = 5 * 1024 * 1024
	}
	if settings.UserAgent == "" {
		settings.UserAgent = "railway-wiki/1.0"
	}
	return &HTTPClient{
		settings: settings,
		client:   &http.Client{Timeout: settings.Timeout},
		cache:    make(map[string]cacheEntry),
		flight:   make(map[string]*inFlight),
		last:     make(map[string]time.Time),
	}, nil
}

func (c *HTTPClient) Search(ctx context.Context, request SearchRequest) ([]Candidate, error) {
	query, err := c.searchQuery(request)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	c.mu.Lock()
	c.pruneExpiredLocked(now)
	if cached, ok := c.cache[query]; ok && cached.expiresAt.After(now) {
		items := cloneCandidates(cached.items)
		c.mu.Unlock()
		return items, nil
	}
	if flight, ok := c.flight[query]; ok {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, classifyContext(ctx.Err())
		case <-flight.done:
			return cloneCandidates(flight.items), flight.err
		}
	}
	caller := request.Caller
	if caller == "" {
		caller = "anonymous"
	}
	if last := c.last[caller]; c.settings.MinInterval > 0 && !last.IsZero() && now.Sub(last) < c.settings.MinInterval {
		c.mu.Unlock()
		return nil, ErrRateLimited
	}
	c.last[caller] = now
	flight := &inFlight{done: make(chan struct{})}
	c.flight[query] = flight
	c.mu.Unlock()

	items, executeErr := c.execute(ctx, query)
	c.mu.Lock()
	flight.items, flight.err = cloneCandidates(items), executeErr
	if executeErr == nil {
		c.cache[query] = cacheEntry{items: cloneCandidates(items), expiresAt: time.Now().Add(c.settings.CacheTTL)}
	}
	delete(c.flight, query)
	close(flight.done)
	c.mu.Unlock()
	return items, executeErr
}

func (c *HTTPClient) pruneExpiredLocked(now time.Time) {
	for query, entry := range c.cache {
		if !entry.expiresAt.After(now) {
			delete(c.cache, query)
		}
	}
}

func (c *HTTPClient) Fetch(ctx context.Context, elementType string, elementID int64) (*Candidate, error) {
	if !validElementType(elementType) || elementID <= 0 {
		return nil, ErrBadRequest
	}
	query := fmt.Sprintf("[out:json][timeout:20];%s(%d);out center tags 1;", elementType, elementID)
	items, err := c.execute(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrBadRequest
	}
	return &items[0], nil
}

func (c *HTTPClient) searchQuery(request SearchRequest) (string, error) {
	if (request.Bounds == nil) == (request.Point == nil) {
		return "", ErrBadRequest
	}
	var selector string
	if b := request.Bounds; b != nil {
		if b.South < -90 || b.North > 90 || b.West < -180 || b.East > 180 ||
			b.South >= b.North || b.West >= b.East || b.North-b.South > 2 || b.East-b.West > 2 {
			return "", ErrBadRequest
		}
		selector = fmt.Sprintf("(%.6f,%.6f,%.6f,%.6f)", b.South, b.West, b.North, b.East)
	} else {
		p := request.Point
		if p.Latitude < -90 || p.Latitude > 90 || p.Longitude < -180 || p.Longitude > 180 ||
			p.RadiusMeters < 1 || p.RadiusMeters > 10000 {
			return "", ErrBadRequest
		}
		selector = fmt.Sprintf("(around:%d,%.6f,%.6f)", p.RadiusMeters, p.Latitude, p.Longitude)
	}
	limit := c.settings.MaxCandidates + 1
	return fmt.Sprintf(`[out:json][timeout:20];(
  nwr["railway"~"^(station|halt|tram_stop)$"]%s;
  nwr["station"~"^(train|subway|light_rail|monorail|tram)$"]%s;
);out center tags %d;`, selector, selector, limit), nil
}

func (c *HTTPClient) execute(ctx context.Context, query string) ([]Candidate, error) {
	ctx, cancel := context.WithTimeout(ctx, c.settings.Timeout)
	defer cancel()
	form := url.Values{"data": []string{query}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.settings.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.settings.UserAgent)
	if c.settings.APIKey != "" {
		req.Header.Set("X-API-Key", c.settings.APIKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, ErrTimeout
		}
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("%w: upstream status %d", ErrUnavailable, resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, c.settings.MaxResponse+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if int64(len(payload)) > c.settings.MaxResponse {
		return nil, ErrTooLarge
	}
	var result overpassResponse
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON", ErrUnavailable)
	}
	items := make([]Candidate, 0, min(len(result.Elements), c.settings.MaxCandidates))
	for _, element := range result.Elements {
		candidate, ok := normalizeElement(element)
		if !ok {
			continue
		}
		items = append(items, candidate)
		if len(items) == c.settings.MaxCandidates {
			break
		}
	}
	return items, nil
}

type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

type overpassElement struct {
	Type   string            `json:"type"`
	ID     int64             `json:"id"`
	Lat    *float64          `json:"lat"`
	Lon    *float64          `json:"lon"`
	Center *elementCenter    `json:"center"`
	Tags   map[string]string `json:"tags"`
}

type elementCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func normalizeElement(element overpassElement) (Candidate, bool) {
	if !validElementType(element.Type) || element.ID <= 0 {
		return Candidate{}, false
	}
	var latitude, longitude float64
	if element.Lat != nil && element.Lon != nil {
		latitude, longitude = *element.Lat, *element.Lon
	} else if element.Center != nil {
		latitude, longitude = element.Center.Lat, element.Center.Lon
	} else {
		return Candidate{}, false
	}
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return Candidate{}, false
	}
	tags := element.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	if !isRailCandidate(tags) {
		return Candidate{}, false
	}
	name := strings.TrimSpace(tags["name"])
	if name == "" {
		name = strings.TrimSpace(tags["ref"])
	}
	if name == "" {
		name = "Unnamed railway station"
	}
	mode := firstNonEmpty(tags["station"], tags["railway"])
	return Candidate{
		ElementType: element.Type,
		ElementID:   element.ID,
		Name:        name,
		NameEn:      stringPointer(tags["name:en"]),
		Ref:         stringPointer(tags["ref"]),
		Railway:     stringPointer(tags["railway"]),
		Mode:        stringPointer(mode),
		Operator:    stringPointer(tags["operator"]),
		Network:     stringPointer(tags["network"]),
		Latitude:    latitude,
		Longitude:   longitude,
		Tags:        tags,
	}, true
}

func isRailCandidate(tags map[string]string) bool {
	railway := strings.TrimSpace(tags["railway"])
	// Some entrance nodes also carry station=subway. They must not be offered
	// as importable stations even though the broader station selector finds
	// them upstream.
	if railway == "entrance" || strings.HasSuffix(railway, "_entrance") {
		return false
	}
	if tags["entrance"] != "" && railway != "station" && railway != "halt" && railway != "tram_stop" {
		return false
	}
	switch railway {
	case "station", "halt", "tram_stop":
		return true
	}
	switch tags["station"] {
	case "train", "subway", "light_rail", "monorail", "tram":
		return true
	default:
		return false
	}
}

func validElementType(value string) bool {
	return value == "node" || value == "way" || value == "relation"
}

func stringPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func classifyContext(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	return err
}

func cloneCandidates(items []Candidate) []Candidate {
	cloned := make([]Candidate, len(items))
	for i, item := range items {
		cloned[i] = item
		cloned[i].Tags = make(map[string]string, len(item.Tags))
		for key, value := range item.Tags {
			cloned[i].Tags[key] = value
		}
	}
	return cloned
}
