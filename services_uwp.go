package main

import (
	"context"
	"fmt"
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

type UwpInstallPlan struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

func (s *VersionService) lookupUwpEntry(updateID string) (*uwpdownload.Version, string) {
	all, err := uwpdownload.LoadCatalog()
	if err != nil {
		return nil, "ERR_UWP_CATALOG"
	}
	for i := range all {
		if strings.EqualFold(all[i].UUID, strings.TrimSpace(updateID)) {
			return &all[i], ""
		}
	}
	return nil, "ERR_UWP_INVALID_VERSION"
}

// UwpPrepareInstall validates everything and resolves a fresh download URL.
// The frontend downloads it with StartFileDownload (progress UI included)
// and then calls UwpFinishInstall. URLs expire, so start the download promptly.
func (s *VersionService) UwpPrepareInstall(updateID string, instanceName string) UwpInstallPlan {
	if msg := mcservice.ValidateVersionFolderName(instanceName); msg != "" {
		return UwpInstallPlan{Error: msg}
	}
	if !registry.IsDevModeEnabled() {
		return UwpInstallPlan{Error: "ERR_UWP_DEV_MODE"}
	}
	entry, errCode := s.lookupUwpEntry(updateID)
	if errCode != "" {
		return UwpInstallPlan{Error: errCode}
	}
	pkg, err := uwpdownload.ResolvePackage(context.Background(), entry.UUID)
	if err != nil {
		return UwpInstallPlan{Error: uwp.ErrorCode(err)}
	}
	filename, err := uwpdownload.Filename(entry.Version, entry.Type)
	if err != nil {
		return UwpInstallPlan{Error: "ERR_UWP_INVALID_VERSION"}
	}
	return UwpInstallPlan{URL: pkg.URL, Filename: filename}
}

// UwpFinishInstall verifies the downloaded file and registers the instance.
func (s *VersionService) UwpFinishInstall(tmpDest string, updateID string, instanceName string) string {
	if msg := mcservice.ValidateVersionFolderName(instanceName); msg != "" {
		return msg
	}
	instanceName = strings.TrimSpace(instanceName)
	if !registry.IsDevModeEnabled() {
		return "ERR_UWP_DEV_MODE"
	}
	entry, errCode := s.lookupUwpEntry(updateID)
	if errCode != "" {
		return errCode
	}
	pkg, err := uwpdownload.ResolvePackage(context.Background(), entry.UUID)
	if err != nil {
		return uwp.ErrorCode(err)
	}
	if err := pkg.Verify(tmpDest); err != nil {
		return "ERR_UWP_INTEGRITY"
	}
	return finishUwpInstall(context.Background(), tmpDest, entry, instanceName)
}

func finishUwpInstall(ctx context.Context, tmpPath string, entry *uwpdownload.Version, instanceName string) string {
	vdir := uwpInstanceDir(instanceName)
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
	instanceName = strings.TrimSpace(instanceName)
	if !registry.IsDevModeEnabled() {
		return "ERR_UWP_DEV_MODE"
	}
	updateID = strings.TrimSpace(updateID)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	entry, errCode := s.lookupUwpEntry(updateID)
	if errCode != "" {
		return errCode
	}
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
	return finishUwpInstall(ctx, tmpPath, entry, instanceName)
}

func (s *VersionService) UwpLaunchVersion(name string) (int, error) {
	if msg := mcservice.ValidateVersionFolderName(name); msg != "" {
		return 0, fmt.Errorf("%s", msg)
	}
	dir := uwpInstanceDir(name)
	return uwp.Launch(context.Background(), dir)
}

func (s *VersionService) UwpUnregisterVersion(name string) string {
	if msg := mcservice.ValidateVersionFolderName(name); msg != "" {
		return msg
	}
	dir := uwpInstanceDir(name)
	if err := uwp.Unregister(context.Background(), dir); err != nil {
		return uwp.ErrorCode(err)
	}
	return ""
}

func uwpInstanceDir(name string) string {
	return filepath.Join(mcservice.GetVersionsDir(), strings.TrimSpace(name))
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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ERR_UWP_DOWNLOAD: HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
