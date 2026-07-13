package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryAddsBackendOwnedUIMetadata(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)
	schema := registry.For("Station", "create")

	ui, ok := schema["x-ui-schema"].(map[string]any)
	require.True(t, ok)
	name, ok := ui["name"].(map[string]any)
	require.True(t, ok)
	options := name["ui:options"].(map[string]any)
	require.Equal(t, "form.station.name", options["accessibility_id"])

	layout, ok := schema["x-ui-layout"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "Basics", layout[0]["title"])
	require.Equal(t, "Location", layout[1]["title"])

	properties := schema["properties"].(map[string]any)
	areaGeo := properties["areaGeo"].(map[string]any)
	require.NotContains(t, areaGeo, "nullable")
	require.NotContains(t, areaGeo, "type")
	variants := areaGeo["anyOf"].([]any)
	require.Equal(t, "object", variants[0].(map[string]any)["type"])
	require.Equal(t, "null", variants[1].(map[string]any)["type"])
}

func TestRegistryMarksRelationsAndMultilineFields(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)
	schema := registry.For("RouteStation", "create")
	ui := schema["x-ui-schema"].(map[string]any)
	station := ui["stationId"].(map[string]any)
	require.Equal(t, "foreign-key", station["ui:widget"])
	options := station["ui:options"].(map[string]any)
	require.Equal(t, "stations", options["endpoint"])
	require.Equal(t, true, options["searchable"])

	// Endpoints whose list handler has no q parameter are marked
	// non-searchable so the picker hides its search bar.
	platformTrack := registry.For("PlatformTrack", "create")
	platformUI := platformTrack["x-ui-schema"].(map[string]any)
	platform := platformUI["platformId"].(map[string]any)
	require.Equal(t, "foreign-key", platform["ui:widget"])
	platformOptions := platform["ui:options"].(map[string]any)
	require.Equal(t, "platforms", platformOptions["endpoint"])
	require.Equal(t, false, platformOptions["searchable"])
}

// effectiveProperty unwraps the anyOf nullable rewrite so assertions can look
// at the branch that carries title/description/enum.
func effectiveProperty(t *testing.T, schema map[string]any, field string) map[string]any {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "schema has no properties")
	prop, ok := properties[field].(map[string]any)
	require.True(t, ok, "missing property %q", field)
	variants, ok := prop["anyOf"].([]any)
	if !ok {
		return prop
	}
	for _, v := range variants {
		m := v.(map[string]any)
		if m["type"] != "null" {
			return m
		}
	}
	t.Fatalf("property %q is anyOf with only null branches", field)
	return nil
}

func TestRegistryServesEnums(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)

	companyType := effectiveProperty(t, registry.For("Company", "create"), "companyType")
	require.ElementsMatch(t,
		[]any{"jr", "major_private", "semi_major", "third_sector", "public", "monorail", "tram", "other"},
		companyType["enum"])

	mediaType := effectiveProperty(t, registry.For("Media", "create"), "mediaType")
	require.ElementsMatch(t, []any{"image", "video", "audio"}, mediaType["enum"])

	entityType := effectiveProperty(t, registry.For("MediaAttachment", "create"), "entityType")
	require.ElementsMatch(t,
		[]any{"station", "platform", "route", "operation_route", "train", "company", "track_segment"},
		entityType["enum"])
}

// TestRegistryServesTitlesAndDescriptions guards that every property of every
// served form schema carries a human-readable title and description.
func TestRegistryServesTitlesAndDescriptions(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)
	for _, resource := range resources {
		for _, action := range []string{"create", "update"} {
			schema := registry.For(resource, action)
			properties, ok := schema["properties"].(map[string]any)
			require.True(t, ok, "%s|%s has no properties", resource, action)
			for field := range properties {
				prop := effectiveProperty(t, schema, field)
				title, _ := prop["title"].(string)
				description, _ := prop["description"].(string)
				require.NotEmpty(t, title, "%s|%s property %q has no title", resource, action, field)
				require.NotEmpty(t, description, "%s|%s property %q has no description", resource, action, field)
			}
		}
	}
}

// TestRegistryUpdateMirrorsCreate spot-checks that the anchored/aliased
// request schemas survive the OpenAPI pipeline identically for both actions.
func TestRegistryUpdateMirrorsCreate(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)
	create := effectiveProperty(t, registry.For("Company", "create"), "companyType")
	update := effectiveProperty(t, registry.For("Company", "update"), "companyType")
	require.Equal(t, create, update)
}
