package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/httpx"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/mcservice"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/registry"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/uwp"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/uwpdownload"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/versions"
)

type UwpPrereqs struct {
	DeveloperMode bool `json:"developerMode"`
}

func (s *VersionService) UwpListVersions(channel string) []uwpdownload.Version {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if !uwpdownload.ValidChannel(channel) {
		return []uwpdownload.Version{}
	}
	all, err := uwpdownload.LoadCatalog()
	if err != nil {
		return []uwpdownload.Version{}
	}
	out := make([]uwpdownload.Version, 0)
	for _, v := range all {
		if v.Type == channel {
			out = append(out, v)
		}
	}
	return out
}

func (s *VersionService) UwpGetPrereqs() UwpPrereqs {
	return UwpPrereqs{DeveloperMode: registry.IsDevModeEnabled()}
}

func (s *VersionService) UwpInstallVersion(updateID string, instanceName string) string {
	if msg := mcservice.ValidateVersionFolderName(instanceName); msg != "" {
		return msg
	}
	if !registry.IsDevModeEnabled() {
		return "ERR_UWP_DEV_MODE"
	}
	updateID = strings.TrimSpace(updateID)
	all, err := uwpdownload.LoadCatalog()
	if err != nil {
		return "ERR_UWP_CATALOG"
	}
	var entry *uwpdownload.Version
	for i := range all {
		if strings.EqualFold(all[i].UUID, updateID) {
			entry = &all[i]
			break
		}
	}
	if entry == nil {
		return "ERR_UWP_INVALID_VERSION"
	}
	ctx := context.Background()
	pkg, err := uwpdownload.ResolvePackage(ctx, entry.UUID)
	if err != nil {
		return uwp.ErrorCode(err)
	}
	filename, err := uwpdownload.Filename(entry.Version, entry.Type)
	if err != nil {
		return "ERR_UWP_INVALID_VERSION"
	}
	tmp, err := os.CreateTemp("", "uwp-download-*_"+filename)
	if err != nil {
		return "ERR_UWP_DOWNLOAD"
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)
	if err := downloadURL(ctx, pkg.URL, tmpPath); err != nil {
		return "ERR_UWP_DOWNLOAD"
	}
	if err := pkg.Verify(tmpPath); err != nil {
		return "ERR_UWP_INTEGRITY"
	}
	vdir := filepath.Join(mcservice.GetVersionsDir(), instanceName)
	manifest, err := uwp.Install(ctx, tmpPath, vdir, uwp.Options{})
	if err != nil {
		return uwp.ErrorCode(err)
	}
	if err := uwp.Register(ctx, vdir); err != nil {
		return uwp.ErrorCode(err)
	}
	meta := versions.VersionMeta{
		Name:            instanceName,
		GameVersion:     manifest.GameVersion(),
		Type:            entry.Type,
		PackageType:     versions.PackageTypeUWP,
		EnableIsolation: true,
		CreatedAt:       time.Now(),
	}
	if err := versions.WriteMeta(vdir, meta); err != nil {
		return "ERR_VERSION_META_WRITE"
	}
	return ""
}

func (s *VersionService) UwpLaunchVersion(name string) (int, error) {
	dir := filepath.Join(mcservice.GetVersionsDir(), strings.TrimSpace(name))
	return uwp.Launch(context.Background(), dir)
}

func (s *VersionService) UwpUnregisterVersion(name string) string {
	dir := filepath.Join(mcservice.GetVersionsDir(), strings.TrimSpace(name))
	if err := uwp.Unregister(context.Background(), dir); err != nil {
		return uwp.ErrorCode(err)
	}
	return ""
}

func downloadURL(ctx context.Context, url string, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpx.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
