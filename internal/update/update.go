package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	json "github.com/goccy/go-json"

	buildcfg "github.com/BedrockNexusLauncher/BedrockNexusLauncher/build"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/apppath"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/config"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/httpx"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/types"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/utils"
	"github.com/Masterminds/semver/v3"
	"github.com/mouuff/go-rocket-update/pkg/provider"
	"github.com/mouuff/go-rocket-update/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/application"
	winreg "golang.org/x/sys/windows/registry"
)

var (
	appVersion = "0.0.10"
	isBeta     = true

	shell32            = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW  = shell32.NewProc("ShellExecuteW")
	swShowNormal       = uintptr(1)
	shellExecuteErrCut = uintptr(32)
)

type AppUpdateProgress struct {
	Phase      string `json:"phase"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
}

type githubRelease struct {
	TagName    string               `json:"tag_name"`
	Body       string               `json:"body"`
	Prerelease bool                 `json:"prerelease"`
	Assets     []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
}

const (
	EventAppUpdateStatus   = "app_update_status"
	EventAppUpdateProgress = "app_update_progress"
	EventAppUpdateError    = "app_update_error"
	updateErrorSeparator   = "::"
)

const (
	updateMetadataTimeout = 20 * time.Second
	updateChecksumTimeout = 20 * time.Second
	updateDownloadTimeout = 20 * time.Minute
)

func Init() {
	v, b := parseVersionAndBetaFromBuildConfig(buildcfg.ConfigYAML)
	if v != "" {
		appVersion = v
	}
	isBeta = b
}

func parseVersionAndBetaFromBuildConfig(src string) (version string, beta bool) {
	s := strings.ReplaceAll(src, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	inInfo := false
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "info:" {
			inInfo = true
			continue
		}
		if inInfo {
			if !strings.HasPrefix(lines[i], " ") && strings.Contains(line, ":") {
				break
			}
			if strings.HasPrefix(line, "version:") {
				if m := regexp.MustCompile(`version:\s*"([^"]+)"`).FindStringSubmatch(line); len(m) == 2 {
					version = m[1]
				} else {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						version = strings.TrimSpace(parts[1])
						version = strings.Trim(version, "\"")
					}
				}
			}
			if strings.HasPrefix(line, "beta:") {
				var s string
				if m := regexp.MustCompile(`beta:\s*"([^"]+)"`).FindStringSubmatch(line); len(m) == 2 {
					s = m[1]
				} else {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						s = strings.TrimSpace(parts[1])
						s = strings.Trim(s, "\"")
					}
				}
				ls := strings.ToLower(strings.TrimSpace(s))
				beta = ls == "true" || ls == "yes" || ls == "on" || ls == "1"
			}
		}
	}
	return version, beta
}

func IsBeta() bool { return isBeta }

func GetAppVersion() string {
	return appVersion
}

func makeUpdateErrorPayload(code string, detail string) string {
	if strings.TrimSpace(detail) == "" {
		return code
	}
	return code + updateErrorSeparator + detail
}

func emitUpdateError(code string, err error) error {
	detail := ""
	if err != nil {
		detail = err.Error()
		log.Printf("update %s: %v", code, err)
	} else {
		log.Printf("update %s", code)
	}
	application.Get().Event.Emit(EventAppUpdateError, makeUpdateErrorPayload(code, detail))
	if err != nil {
		return fmt.Errorf("%s: %w", code, err)
	}
	return errors.New(code)
}

func newTimedRequest(parent context.Context, method string, target string, timeout time.Duration) (*http.Request, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	httpx.ApplyDefaultHeaders(req)
	return req, cancel, nil
}

func getReleaseAPIURLs() []string {
	// Release metadata is the root of trust for the whole self-update flow:
	// it decides whether an update exists and which binary gets executed.
	// It must only ever come from GitHub itself. Third-party proxy mirrors
	// may help availability, but they must never be an authority — a
	// compromised mirror could fabricate a release and trigger installation
	// of an attacker-controlled binary.
	return []string{
		"https://api.github.com/repos/BedrockNexusLauncher/BedrockNexusLauncher/releases",
	}
}

func fetchReleases(ctx context.Context) ([]githubRelease, error) {
	apis := getReleaseAPIURLs()
	var lastErr error
	for i := 0; i < len(apis); i++ {
		target := apis[i]
		req, cancel, err := newTimedRequest(ctx, http.MethodGet, target, updateMetadataTimeout)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := httpx.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			cancel()
			lastErr = fmt.Errorf("HTTP %s", resp.Status)
			continue
		}
		var releases []githubRelease
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&releases)
		resp.Body.Close()
		cancel()
		if err != nil {
			lastErr = err
			continue
		}
		if len(releases) > 0 {
			return releases, nil
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no release metadata available")
	}
	return nil, lastErr
}

func getAcceptBeta() bool {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("config.Load failed while checking updates: %v", err)
	}
	return cfg.EnableBetaUpdates || isBeta
}

func selectUpdateRelease(releases []githubRelease, version string) (githubRelease, bool) {
	acceptBeta := getAcceptBeta()
	cur := strings.TrimPrefix(strings.TrimSpace(version), "v")
	vCur, err1 := semver.NewVersion(cur)
	for _, release := range releases {
		if release.Prerelease && !acceptBeta {
			continue
		}
		tag := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
		vRel, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}
		if err1 != nil {
			if release.TagName != version {
				return release, true
			}
			continue
		}
		if vRel.GreaterThan(vCur) {
			return release, true
		}
	}
	return githubRelease{}, false
}

func findAssetByName(release githubRelease, name string) *githubReleaseAsset {
	for i := range release.Assets {
		if strings.EqualFold(strings.TrimSpace(release.Assets[i].Name), strings.TrimSpace(name)) {
			return &release.Assets[i]
		}
	}
	return nil
}

func expectedZipAssetName() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("BedrockNexus_windows_%s.zip", runtime.GOARCH)
	}
	return fmt.Sprintf("BedrockNexus_%s_%s.zip", runtime.GOOS, runtime.GOARCH)
}

func buildAssetCandidateURLs(target string) []string {
	// Direct URL only. Release assets are executable payloads (or the
	// checksums that vouch for them), so — like the release metadata —
	// they are never fetched through third-party proxy mirrors.
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	return []string{target}
}

// validateReleaseAssetURL ensures a release asset is downloaded from
// GitHub itself: the release page host or a GitHub content host.
func validateReleaseAssetURL(raw string) error {
	u, err := neturl.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid asset URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("asset URL must be https: %q", raw)
	}
	if u.User != nil {
		return fmt.Errorf("asset URL must not contain credentials: %q", raw)
	}
	host := strings.ToLower(u.Hostname())
	if host == "github.com" || host == "githubusercontent.com" || strings.HasSuffix(host, ".githubusercontent.com") {
		return nil
	}
	return fmt.Errorf("asset URL host is not a GitHub host: %q", host)
}

// parseSHA256ForAsset extracts the expected SHA-256 for one asset from
// checksum-file content. It accepts both a bare hash and the standard
// "<hash>  <filename>" lines produced by sha256sum; when several files
// are listed, the line naming assetName wins.
func parseSHA256ForAsset(content string, assetName string) (string, error) {
	var single string
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		hash := fields[0]
		if len(hash) != 64 {
			continue
		}
		if _, err := hex.DecodeString(hash); err != nil {
			continue
		}
		name := ""
		if len(fields) > 1 {
			name = strings.TrimPrefix(fields[1], "*")
			name = path.Base(name)
		}
		if name == assetName {
			return strings.ToLower(hash), nil
		}
		if single == "" {
			single = strings.ToLower(hash)
		}
	}
	if single != "" {
		return single, nil
	}
	return "", fmt.Errorf("no sha256 checksum found for %s", assetName)
}

// expectedUpdateSHA256 resolves the trusted SHA-256 for the update ZIP:
// first the release's "<zip>.sha256" checksum asset, then the digest
// GitHub itself reports for the asset. No checksum, no update.
func expectedUpdateSHA256(ctx context.Context, release githubRelease, zipAsset *githubReleaseAsset) (string, error) {
	if sumAsset := findAssetByName(release, zipAsset.Name+".sha256"); sumAsset != nil {
		if err := validateReleaseAssetURL(sumAsset.BrowserDownloadURL); err != nil {
			return "", err
		}
		text, err := downloadTextWithCandidates(ctx, buildAssetCandidateURLs(sumAsset.BrowserDownloadURL), updateChecksumTimeout)
		if err != nil {
			return "", err
		}
		return parseSHA256ForAsset(text, zipAsset.Name)
	}
	if d := strings.TrimSpace(zipAsset.Digest); strings.HasPrefix(strings.ToLower(d), "sha256:") {
		return parseSHA256Checksum(strings.TrimPrefix(strings.ToLower(d), "sha256:"))
	}
	return "", fmt.Errorf("release %s publishes no sha256 checksum for %s", release.TagName, zipAsset.Name)
}

func downloadTextWithCandidates(ctx context.Context, urls []string, timeout time.Duration) (string, error) {
	var lastErr error
	for _, target := range urls {
		req, cancel, err := newTimedRequest(ctx, http.MethodGet, target, timeout)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := httpx.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			cancel()
			lastErr = fmt.Errorf("HTTP %s", resp.Status)
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if err != nil {
			lastErr = err
			continue
		}
		return string(data), nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("download text failed")
	}
	return "", lastErr
}

func parseSHA256Checksum(content string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty checksum content")
	}
	hash := strings.TrimSpace(fields[0])
	if len(hash) != 64 {
		return "", fmt.Errorf("invalid sha256 length")
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return "", err
	}
	return strings.ToLower(hash), nil
}

func calculateSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifyFileSHA256(path string, expected string) error {
	actual, err := calculateSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(expected)) {
		return fmt.Errorf("sha256 mismatch: expected %s got %s", expected, actual)
	}
	return nil
}

func downloadUpdateAsset(ctx context.Context, urls []string, dest string) error {
	var lastErr error
	tmp := dest + ".part"
	for _, target := range urls {
		_ = os.Remove(tmp)
		req, cancel, err := newTimedRequest(ctx, http.MethodGet, target, updateDownloadTimeout)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Accept", "application/zip")
		resp, err := httpx.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			cancel()
			lastErr = fmt.Errorf("HTTP %s", resp.Status)
			continue
		}
		application.Get().Event.Emit(EventAppUpdateStatus, "downloading")
		f, ferr := os.Create(tmp)
		if ferr != nil {
			resp.Body.Close()
			cancel()
			lastErr = ferr
			continue
		}
		var downloaded int64
		total := resp.ContentLength
		buf := make([]byte, 32*1024)
		lastEmit := time.Now().Add(-time.Second)
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				wn, werr := f.Write(buf[:n])
				if werr != nil {
					ferr = werr
					break
				}
				downloaded += int64(wn)
				if time.Since(lastEmit) >= 250*time.Millisecond {
					application.Get().Event.Emit(EventAppUpdateProgress, AppUpdateProgress{Phase: "download", Downloaded: downloaded, Total: total})
					lastEmit = time.Now()
				}
			}
			if rerr != nil {
				if rerr == io.EOF {
					break
				}
				ferr = rerr
				break
			}
		}
		application.Get().Event.Emit(EventAppUpdateProgress, AppUpdateProgress{Phase: "download", Downloaded: downloaded, Total: total})
		cerr := f.Close()
		resp.Body.Close()
		cancel()
		if ferr != nil || cerr != nil {
			_ = os.Remove(tmp)
			if ferr != nil {
				lastErr = ferr
			} else {
				lastErr = cerr
			}
			continue
		}
		if err := os.Rename(tmp, dest); err != nil {
			_ = os.Remove(tmp)
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("download failed")
	}
	return lastErr
}

func startSelfUpdateElevated(exePath string, curVer string) error {
	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(exePath)
	if err != nil {
		return err
	}
	arg := "--self-update=" + strings.TrimPrefix(strings.TrimSpace(curVer), "v")
	args, err := syscall.UTF16PtrFromString(arg)
	if err != nil {
		return err
	}
	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(args)),
		0,
		swShowNormal,
	)
	if ret <= shellExecuteErrCut {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return fmt.Errorf("ShellExecuteW failed with code %d", ret)
	}
	return nil
}

func CheckUpdate(version string) types.CheckUpdate {
	releases, err := fetchReleases(context.Background())
	if err != nil || len(releases) == 0 {
		if err != nil {
			log.Printf("CheckUpdate failed: %v", err)
		}
		return types.CheckUpdate{IsUpdate: false, Version: version}
	}
	release, ok := selectUpdateRelease(releases, version)
	if ok {
		return types.CheckUpdate{IsUpdate: true, Version: release.TagName, Body: release.Body}
	}

	return types.CheckUpdate{IsUpdate: false, Version: version}
}

func Update(version string) error {
	ctx := context.Background()
	application.Get().Event.Emit(EventAppUpdateStatus, "checking")
	exePath, _ := os.Executable()
	instDir := filepath.Dir(exePath)

	cur := strings.TrimSpace(version)
	if cur == "" {
		cur = GetAppVersion()
	}
	releases, err := fetchReleases(ctx)
	if err != nil {
		return emitUpdateError("ERR_UPDATE_RELEASE_UNAVAILABLE", err)
	}
	release, ok := selectUpdateRelease(releases, cur)
	target := strings.TrimSpace(release.TagName)
	if !ok || target == "" {
		return emitUpdateError("ERR_UPDATE_NOT_AVAILABLE", fmt.Errorf("no update available"))
	}
	ver := target
	if !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}
	curVer := cur
	if !strings.HasPrefix(curVer, "v") {
		curVer = "v" + curVer
	}

	if runtime.GOOS == "windows" && !utils.CanWriteDir(instDir) {
		application.Get().Event.Emit(EventAppUpdateStatus, "elevating")
		if err := startSelfUpdateElevated(exePath, curVer); err != nil {
			return emitUpdateError("ERR_UPDATE_ADMIN_REQUIRED", err)
		}
		os.Exit(0)
		return nil
	}

	archName := fmt.Sprintf("%s.zip", ver)

	execName := "BedrockNexus"
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}

	base := apppath.BaseRoot()
	updDir := filepath.Join(base, "updates")
	if err := os.MkdirAll(updDir, 0o755); err != nil {
		return emitUpdateError("ERR_UPDATE_WRITE_FAILED", err)
	}
	zipPath := filepath.Join(updDir, archName)

	// The update ZIP is executed, so it is only ever taken from the
	// release's own GitHub asset and only installed after its SHA-256
	// matches the checksum published with the release. A ZIP left over
	// in the updates folder is never trusted on its own: it is reused
	// only when it verifies against the freshly fetched release.
	zipAsset := findAssetByName(release, expectedZipAssetName())
	if zipAsset == nil {
		return emitUpdateError("ERR_UPDATE_ASSET_MISSING", fmt.Errorf("release %s has no %s asset", target, expectedZipAssetName()))
	}
	if err := validateReleaseAssetURL(zipAsset.BrowserDownloadURL); err != nil {
		return emitUpdateError("ERR_UPDATE_VERIFY_FAILED", err)
	}
	expectedSHA, err := expectedUpdateSHA256(ctx, release, zipAsset)
	if err != nil {
		return emitUpdateError("ERR_UPDATE_VERIFY_FAILED", err)
	}

	verified := false
	if fi, statErr := os.Stat(zipPath); statErr == nil && fi.Size() > 0 {
		if err := verifyFileSHA256(zipPath, expectedSHA); err == nil {
			verified = true
			application.Get().Event.Emit(EventAppUpdateStatus, "downloaded")
			application.Get().Event.Emit(EventAppUpdateProgress, AppUpdateProgress{Phase: "download", Downloaded: fi.Size(), Total: fi.Size()})
		} else {
			_ = os.Remove(zipPath)
		}
	}
	if !verified {
		if err := downloadUpdateAsset(ctx, buildAssetCandidateURLs(zipAsset.BrowserDownloadURL), zipPath); err != nil {
			return emitUpdateError("ERR_UPDATE_DOWNLOAD_FAILED", err)
		}
		if err := verifyFileSHA256(zipPath, expectedSHA); err != nil {
			_ = os.Remove(zipPath)
			return emitUpdateError("ERR_UPDATE_VERIFY_FAILED", err)
		}
	}

	u := &updater.Updater{
		Provider:       &provider.Zip{Path: zipPath},
		ExecutableName: execName,
		Version:        curVer,
	}

	application.Get().Event.Emit(EventAppUpdateStatus, "installing")
	status, err := u.Update()
	if err != nil {
		return emitUpdateError("ERR_UPDATE_INSTALL_FAILED", err)
	}

	_ = u.CleanUp()
	_ = os.RemoveAll(updDir)
	log.Printf("Update status: %+v", status)
	application.Get().Event.Emit(EventAppUpdateStatus, "installed")

	exe, err := os.Executable()
	if err != nil {
		return emitUpdateError("ERR_UPDATE_RESTART_FAILED", err)
	}
	application.Get().Event.Emit(EventAppUpdateStatus, "restarting")
	if err := restartProgram(exe); err != nil {
		return emitUpdateError("ERR_UPDATE_RESTART_FAILED", err)
	}
	return nil
}

func restartProgram(exePath string) error {
	cmd := exec.Command(exePath, "--post-update-restart")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart program: %w", err)
	}
	os.Exit(0)
	return nil
}

func isZhCnLocaleValue(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	if s == "" {
		return false
	}
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.TrimPrefix(s, "0x")
	if strings.HasPrefix(s, "zh-cn") || s == "zh" || s == "zh-hans" {
		return true
	}
	if strings.Contains(s, "chinese (simplified") {
		return true
	}
	return s == "804" || s == "0804" || s == "00000804"
}

func isWindowsTimeZoneChina() bool {
	k, err := winreg.OpenKey(winreg.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\TimeZoneInformation`, winreg.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	for _, field := range []string{"TimeZoneKeyName", "StandardName"} {
		if val, _, err := k.GetStringValue(field); err == nil {
			if strings.EqualFold(strings.TrimSpace(val), "China Standard Time") {
				return true
			}
		}
	}
	return false
}

func isWindowsLocaleZhCN() bool {
	k, err := winreg.OpenKey(winreg.CURRENT_USER, `Control Panel\International`, winreg.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		for _, field := range []string{"LocaleName", "Locale", "sLanguage"} {
			if val, _, err := k.GetStringValue(field); err == nil && isZhCnLocaleValue(val) {
				return true
			}
		}
	}
	envs := []string{"LANG", "LANGUAGE", "LC_ALL", "LC_MESSAGES"}
	for _, key := range envs {
		if isZhCnLocaleValue(os.Getenv(key)) {
			return true
		}
	}
	return false
}

func isChinaUser() bool {
	if runtime.GOOS == "windows" {
		return isWindowsTimeZoneChina() || isWindowsLocaleZhCN()
	}

	isTz := false
	isLang := false
	if b, err := os.ReadFile("/etc/timezone"); err == nil {
		s := strings.ToLower(strings.TrimSpace(string(b)))
		if s == "asia/shanghai" || s == "asia/urumqi" {
			isTz = true
		}
	} else {
		if p, e := os.Readlink("/etc/localtime"); e == nil {
			low := strings.ToLower(p)
			if strings.Contains(low, "asia/shanghai") || strings.Contains(low, "asia/urumqi") {
				isTz = true
			}
		}
	}
	envs := []string{"LANG", "LANGUAGE", "LC_ALL", "LC_MESSAGES"}
	for _, k := range envs {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
		if v == "" {
			continue
		}
		if strings.HasPrefix(v, "zh_cn") || strings.HasPrefix(v, "zh-cn") || v == "zh" {
			isLang = true
			break
		}
	}
	return isTz || isLang
}
