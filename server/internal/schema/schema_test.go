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
	require.Equal(t, "relation", station["ui:widget"])
	require.Equal(t, "stations", station["ui:options"].(map[string]any)["resource"])
}
