package mcservice

import (
	"context"
	"path/filepath"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/config"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/gdkversions"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/httpx"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/xbox"
)

// FetchHistoricalVersionsWithStore supplements the historical GDK catalog with
// live Xbox Store discovery (signed-in users), keeping newly discovered versions
// across refreshes until the catalog includes them. Falls back to the plain
// historical fetch (with the extra CN mirror) when discovery yields nothing.
func FetchHistoricalVersionsWithStore(preferCN bool) map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := gdkversions.NewClient(httpx.NewClient(10*time.Second), xbox.GetPackageUpdateAuthorization)
	catalog := client.Fetch(ctx, preferCN, filepath.Join(config.ConfigDir(), "gdk-pending.json"))
	if len(catalog.ReleaseVersions)+len(catalog.PreviewVersions) > 0 {
		return gdkCatalogToMap(catalog)
	}
	return FetchHistoricalVersions(preferCN)
}

func gdkCatalogToMap(catalog gdkversions.Catalog) map[string]interface{} {
	release := make([]map[string]interface{}, 0, len(catalog.ReleaseVersions))
	for _, v := range catalog.ReleaseVersions {
		release = append(release, map[string]interface{}{
			"version":   v.Version,
			"urls":      v.URLs,
			"timestamp": v.Timestamp,
		})
	}
	preview := make([]map[string]interface{}, 0, len(catalog.PreviewVersions))
	for _, v := range catalog.PreviewVersions {
		preview = append(preview, map[string]interface{}{
			"version":   v.Version,
			"urls":      v.URLs,
			"timestamp": v.Timestamp,
		})
	}
	obj := map[string]interface{}{
		"releaseVersions": release,
		"previewVersions": preview,
	}
	if catalog.Source != "" {
		obj["_source"] = catalog.Source
	}
	return obj
}
