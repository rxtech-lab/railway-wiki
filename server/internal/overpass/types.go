// Package overpass provides a bounded, railway-specific client for the
// self-hosted Overpass API. It intentionally does not expose arbitrary QL.
package overpass

import (
	"context"
	"errors"
	"time"
)

var (
	ErrBadRequest  = errors.New("invalid overpass search")
	ErrUnavailable = errors.New("overpass unavailable")
	ErrTimeout     = errors.New("overpass timeout")
	ErrTooLarge    = errors.New("overpass response too large")
	ErrRateLimited = errors.New("overpass search rate limited")
)

// Bounds describes a non-dateline-crossing map viewport.
type Bounds struct {
	South float64
	West  float64
	North float64
	East  float64
}

// PointSearch describes a bounded radial search.
type PointSearch struct {
	Latitude     float64
	Longitude    float64
	RadiusMeters int
}

// SearchRequest permits exactly one bounds or point query. Caller is used only
// for per-admin throttling and is never sent upstream.
type SearchRequest struct {
	Bounds *Bounds
	Point  *PointSearch
	Caller string
}

// Candidate is the normalized subset of OSM railway data consumed by clients.
type Candidate struct {
	ElementType string            `json:"elementType"`
	ElementID   int64             `json:"elementId"`
	Name        string            `json:"name"`
	NameEn      *string           `json:"nameEn,omitempty"`
	Ref         *string           `json:"ref,omitempty"`
	Railway     *string           `json:"railway,omitempty"`
	Mode        *string           `json:"mode,omitempty"`
	Operator    *string           `json:"operator,omitempty"`
	Network     *string           `json:"network,omitempty"`
	Latitude    float64           `json:"latitude"`
	Longitude   float64           `json:"longitude"`
	Tags        map[string]string `json:"tags"`
}

type Settings struct {
	URL           string
	APIKey        string
	Timeout       time.Duration
	CacheTTL      time.Duration
	MaxCandidates int
	MaxResponse   int64
	MinInterval   time.Duration
	UserAgent     string
}

// Client is deliberately narrow so handlers cannot become a generic query
// proxy. Fetch refetches one selected OSM element before import.
type Client interface {
	Search(context.Context, SearchRequest) ([]Candidate, error)
	Fetch(context.Context, string, int64) (*Candidate, error)
}
