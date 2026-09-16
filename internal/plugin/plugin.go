// Package plugin implements the file-based plugin host for Bedrock Nexus.
//
// Discovery contract: each plugin lives at <PluginsDir>/<Name>/ with a
// manifest at <PluginsDir>/<Name>/manifest/manifest.json. The launcher
// watches PluginsDir and treats a plugin as present when its manifest is
// valid JSON with the required fields. No code changes are needed to add
// a plugin — dropping the folder in is enough.
package plugin

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	errMissingField  = errors.New("plugin manifest missing required field")
	errNotPrivate    = errors.New("plugin manifest must be private")
	errBadPermission = errors.New("plugin manifest requests forbidden permission")
)

// UiPlacement lets a plugin declare where its UI surfaces inside the
// launcher (e.g. settings > others). Generic contract — any plugin can use it.
type UiPlacement struct {
	Section            string `json:"section"`
	Tab                string `json:"tab"`
	Order              int    `json:"order"`
	VisibleWhenPresent bool   `json:"visibleWhenPresent"`
	LabelKey           string `json:"labelKey"`
	DescKey            string `json:"descKey"`
}

// Manifest is the plugin recognition contract.
type Manifest struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	Entry       string      `json:"entry"`
	Description string      `json:"description"`
	Author      string      `json:"author"`
	Icon        string      `json:"icon"`
	Permissions []string    `json:"permissions"`
	Private     bool        `json:"private"`
	UiPlacement UiPlacement `json:"uiPlacement"`
}

// PluginsDir resolves the plugin root. Candidates in order:
// 1. <executable dir>/plugins (production / installed layout)
// 2. <cwd>/plugins (dev / repo root layout)
// Returns the first candidate that exists; otherwise the exe-side path.
func PluginsDir() string {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "plugins"))
	}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		candidates = append(candidates, filepath.Join(cwd, "plugins"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return "plugins"
}

// PluginDir returns the absolute dir of a plugin by folder name.
func PluginDir(name string) string {
	return filepath.Join(PluginsDir(), name)
}

// ManifestPath returns the manifest file path for a plugin.
func ManifestPath(name string) string {
	return filepath.Join(PluginDir(name), "manifest", "manifest.json")
}

// ReadManifest parses and validates the manifest of a plugin.
// Validation is intentionally shallow: required fields present, private true,
// no network permissions.
func ReadManifest(name string) (*Manifest, error) {
	raw, err := os.ReadFile(ManifestPath(name))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if strings.TrimSpace(m.ID) == "" || strings.TrimSpace(m.Name) == "" ||
		strings.TrimSpace(m.Version) == "" || strings.TrimSpace(m.Entry) == "" {
		return nil, errMissingField
	}
	if !m.Private {
		return nil, errNotPrivate
	}
	for _, p := range m.Permissions {
		lp := strings.ToLower(strings.TrimSpace(p))
		if strings.Contains(lp, "http") || strings.Contains(lp, "network") ||
			strings.Contains(lp, "socket") || strings.Contains(lp, "publish") ||
			strings.Contains(lp, "telemetry") {
			return nil, errBadPermission
		}
	}
	return &m, nil
}

// IsPresent reports whether a plugin is discoverable (valid manifest).
func IsPresent(name string) bool {
	_, err := ReadManifest(name)
	return err == nil
}

// PatchPluginName is the folder name of the personal patch plugin.
const PatchPluginName = "Patch"

// PatchDir returns the on-disk dir of the patch plugin.
func PatchDir() string {
	return PluginDir(PatchPluginName)
}

// IsPatchPresent reports whether the patch plugin is installed.
func IsPatchPresent() bool {
	return IsPresent(PatchPluginName)
}
