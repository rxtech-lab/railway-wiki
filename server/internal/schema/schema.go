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
	"sort"
	"strings"
	"unicode"

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
			applyUIMetadata(res, m)
			reg.byKey[res+"|"+action] = m
		}
	}
	return reg, nil
}

func applyUIMetadata(resource string, schema map[string]any) {
	properties, _ := schema["properties"].(map[string]any)
	fields := make([]string, 0, len(properties))
	for field := range properties {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	fields = preferredOrder(resource, fields)

	uiSchema := map[string]any{"ui:order": fields}
	for _, field := range fields {
		configuration := map[string]any{
			"ui:options": map[string]any{
				"accessibility_id": "form." + kebab(resource) + "." + field,
			},
		}
		switch {
		case strings.HasSuffix(field, "Id"):
			configuration["ui:widget"] = "relation"
			configuration["ui:options"].(map[string]any)["resource"] = relationResource(field)
		case field == "areaGeo" || field == "geo":
			configuration["ui:widget"] = "geojson"
		case field == "latitude" || field == "longitude":
			configuration["ui:widget"] = "coordinate"
		case field == "description" || field == "note":
			configuration["ui:widget"] = "textarea"
		case resource == "Media" && field == "url":
			configuration["ui:widget"] = "media-upload"
		}
		uiSchema[field] = configuration
	}
	schema["x-ui-schema"] = uiSchema
	schema["x-ui-layout"] = layout(resource, fields)
}

func preferredOrder(resource string, available []string) []string {
	if resource != "Station" {
		return available
	}
	preferred := []string{"name", "nameEn", "stationNumber", "description", "latitude", "longitude", "areaGeo", "openedAt", "closedAt"}
	known := make(map[string]bool, len(available))
	for _, value := range available {
		known[value] = true
	}
	result := make([]string, 0, len(available))
	for _, value := range preferred {
		if known[value] {
			result = append(result, value)
			delete(known, value)
		}
	}
	for _, value := range available {
		if known[value] {
			result = append(result, value)
		}
	}
	return result
}

func layout(resource string, fields []string) []map[string]any {
	if resource != "Station" {
		return []map[string]any{{"id": "details", "title": "Details", "fields": fields}}
	}
	return []map[string]any{
		{"id": "basics", "title": "Basics", "fields": present(fields, "name", "nameEn", "stationNumber", "description")},
		{"id": "location", "title": "Location", "fields": present(fields, "latitude", "longitude", "areaGeo")},
		{"id": "operations", "title": "Operations", "fields": present(fields, "openedAt", "closedAt")},
		{"id": "review", "title": "Review", "fields": []string{}},
	}
}

func present(fields []string, desired ...string) []string {
	available := make(map[string]bool, len(fields))
	for _, field := range fields {
		available[field] = true
	}
	result := make([]string, 0, len(desired))
	for _, field := range desired {
		if available[field] {
			result = append(result, field)
		}
	}
	return result
}

func relationResource(field string) string {
	base := strings.TrimSuffix(field, "Id")
	switch base {
	case "company":
		return "companies"
	case "station", "fromStation", "toStation", "originStation", "destinationStation":
		return "stations"
	case "route":
		return "routes"
	case "operationRoute":
		return "operation-routes"
	case "calendar":
		return "service-calendars"
	case "timetableVersion":
		return "timetable-versions"
	case "train":
		return "trains"
	case "trainRun":
		return "train-runs"
	case "platform":
		return "platforms"
	case "trackSegment":
		return "track-segments"
	case "media":
		return "media"
	default:
		return kebab(base) + "s"
	}
}

func kebab(value string) string {
	var result strings.Builder
	for index, character := range value {
		if unicode.IsUpper(character) && index > 0 {
			result.WriteByte('-')
		}
		result.WriteRune(unicode.ToLower(character))
	}
	return result.String()
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
// draft-2020-12. Nullable schemas use anyOf instead of a type array because the
// iOS JSONSchema decoder accepts only a single string in the type keyword.
func normalize(node any) {
	switch v := node.(type) {
	case map[string]any:
		nullable, _ := v["nullable"].(bool)
		delete(v, "nullable")
		for _, child := range v {
			normalize(child)
		}
		if nullable {
			wrapNullable(v)
		}
	case []any:
		for _, child := range v {
			normalize(child)
		}
	}
}

func wrapNullable(schema map[string]any) {
	nonNull := make(map[string]any, len(schema))
	for key, value := range schema {
		nonNull[key] = value
		delete(schema, key)
	}
	schema["anyOf"] = []any{
		nonNull,
		map[string]any{"type": "null"},
	}
}
