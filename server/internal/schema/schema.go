// Package schema serves JSON Schema (draft 2020-12) documents for the management
// create/update request bodies, so the frontend can render and validate forms.
//
// The OpenAPI spec is embedded (a copy is placed here by `make generate`) and
// parsed once at startup; each resource's Create/Update request-body component is
// converted from OpenAPI 3.0 schema form to JSON Schema draft 2020-12.
package schema

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed openapi.yaml
var specBytes []byte

const draft2020 = "https://json-schema.org/draft/2020-12/schema"

// resources are the management resources exposing a schema endpoint. Component
// names follow the Create<Resource>Request / Update<Resource>Request convention.
var resources = []string{
	"Company", "Station", "StationCode", "StationTransfer", "Platform",
	"Route", "RouteCompany", "RouteStation", "TrackSegment", "PlatformTrack",
	"OperationRoute", "OperationRouteCompany", "OperationRouteSection", "OperationRouteStop",
	"TimetableVersion", "ServiceCalendar", "ServiceCalendarException",
	"Train", "TrainRun", "TrainRunStop", "Media", "MediaAttachment",
}

// Registry maps "<Resource>|<action>" to a ready-to-serve JSON Schema document.
type Registry struct {
	byKey map[string]map[string]any
}

// NewRegistry parses the embedded spec and builds the schema documents.
func NewRegistry() (*Registry, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(specBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded openapi spec: %w", err)
	}

	reg := &Registry{byKey: make(map[string]map[string]any, len(resources)*2)}
	for _, res := range resources {
		for action, comp := range map[string]string{
			"create": "Create" + res + "Request",
			"update": "Update" + res + "Request",
		} {
			ref := doc.Components.Schemas[comp]
			if ref == nil || ref.Value == nil {
				return nil, fmt.Errorf("missing component schema %q", comp)
			}
			m, err := toJSONSchema(ref.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to convert schema %q: %w", comp, err)
			}
			reg.byKey[res+"|"+action] = m
		}
	}
	return reg, nil
}

// For returns the JSON Schema document for a resource + action, or an empty
// object schema when unknown.
func (r *Registry) For(resource, action string) map[string]any {
	if m, ok := r.byKey[resource+"|"+action]; ok {
		return m
	}
	return map[string]any{"$schema": draft2020, "type": "object"}
}

// toJSONSchema marshals an OpenAPI 3.0 schema and normalizes it to draft 2020-12.
func toJSONSchema(s *openapi3.Schema) (map[string]any, error) {
	raw, err := s.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	normalize(m)
	m["$schema"] = draft2020
	return m, nil
}

// normalize recursively rewrites OpenAPI 3.0 constructs into JSON Schema
// draft-2020-12: `nullable: true` becomes a "null" member of `type`.
func normalize(node any) {
	switch v := node.(type) {
	case map[string]any:
		if nullable, ok := v["nullable"].(bool); ok {
			delete(v, "nullable")
			if nullable {
				v["type"] = withNull(v["type"])
			}
		}
		for _, child := range v {
			normalize(child)
		}
	case []any:
		for _, child := range v {
			normalize(child)
		}
	}
}

// withNull adds "null" to a JSON Schema type, which may be a string or a list.
func withNull(t any) any {
	switch tv := t.(type) {
	case string:
		return []any{tv, "null"}
	case []any:
		for _, e := range tv {
			if s, ok := e.(string); ok && s == "null" {
				return tv
			}
		}
		return append(tv, "null")
	default:
		return "null"
	}
}
