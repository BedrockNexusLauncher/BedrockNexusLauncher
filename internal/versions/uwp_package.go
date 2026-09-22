package versions

import (
	"strconv"
	"strings"
)

// UWP package-type taxonomy mirrored from LeviLauncher v1.0.x
// (internal/versions/meta.go + editor.go + version_compare.go).
// Kept in this file so local meta.go (Bedrock Nexus fields) stays untouched.

const (
	PackageTypeGDK = "gdk"
	PackageTypeUWP = "uwp"
)

// NormalizePackageType keeps metadata written before UWP support compatible.
func NormalizePackageType(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), PackageTypeUWP) {
		return PackageTypeUWP
	}
	return PackageTypeGDK
}

// Editor was Preview-only from 1.19.80.20 and reached retail in 1.21.50.
const (
	EditorReleaseMinVersion = "1.21.50"
	EditorPreviewMinVersion = "1.19.80.20"
)

func EditorMinVersion(channel string) string {
	if strings.EqualFold(strings.TrimSpace(channel), "preview") {
		return EditorPreviewMinVersion
	}
	return EditorReleaseMinVersion
}

func SupportsEditorMode(gameVersion, channel string) bool {
	return gameVersionAtLeast(gameVersion, EditorMinVersion(channel))
}

func EditorLaunchURI(packageType, channel string) string {
	protocol := "minecraft:"
	if strings.EqualFold(strings.TrimSpace(channel), "preview") {
		protocol = "minecraft-preview:"
	}
	if NormalizePackageType(packageType) == PackageTypeUWP {
		return protocol + "?Editor=true"
	}
	return protocol + "//creator/?Editor=true"
}

func parseNumericGameVersion(value string) ([4]uint64, bool) {
	var version [4]uint64
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) > len(version) {
		return version, false
	}
	for i, part := range parts {
		if part == "" || strings.IndexFunc(part, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			return version, false
		}
		number, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return version, false
		}
		version[i] = number
	}
	return version, true
}

func gameVersionAtLeast(value, minimum string) bool {
	version, valid := parseNumericGameVersion(value)
	minVersion, minValid := parseNumericGameVersion(minimum)
	if !valid || !minValid {
		return false
	}
	for i, part := range version {
		if part != minVersion[i] {
			return part > minVersion[i]
		}
	}
	return true
}
