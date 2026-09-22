package mcservice

import (
	"testing"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/gdkversions"
)

func TestGdkCatalogToMapKeepsShape(t *testing.T) {
	catalog := gdkversions.Catalog{
		FileVersion: 1,
		ReleaseVersions: []gdkversions.Version{
			{Version: "Release 1.21.93.1", URLs: []string{"https://cdn.example/a"}, Timestamp: 123},
		},
		Source: "https://packagespc.xboxlive.com",
	}
	obj := gdkCatalogToMap(catalog)
	rel, ok := obj["releaseVersions"].([]map[string]interface{})
	if !ok || len(rel) != 1 {
		t.Fatalf("expected 1 release entry, got %#v", obj["releaseVersions"])
	}
	if rel[0]["version"] != "Release 1.21.93.1" {
		t.Fatalf("unexpected version: %#v", rel[0])
	}
	if obj["_source"] != "https://packagespc.xboxlive.com" {
		t.Fatalf("unexpected source: %#v", obj["_source"])
	}
	prev, ok := obj["previewVersions"].([]map[string]interface{})
	if !ok || len(prev) != 0 {
		t.Fatalf("expected empty preview list, got %#v", obj["previewVersions"])
	}
}
