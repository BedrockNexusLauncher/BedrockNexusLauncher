package patch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/apppath"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/config"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/gdk"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/mcservice"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/plugin"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/registry"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/utils"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/versions"
	"golang.org/x/sys/windows"
)

const (
	patchTempDirPrefix    = "BedrockNexus_Patch_"
	patchScriptName       = "patch.bat"
	patchRunnerName       = "run_patch.cmd"
	patchExtractionMaxAge = 24 * time.Hour
)

var (
	shell32           = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

func ResolveDir() (string, error) {
	dir := plugin.PatchDir()
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", fmt.Errorf("patch plugin not installed at %s", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, patchScriptName)); err != nil {
		return "", fmt.Errorf("patch script not found in plugin: %w", err)
	}
	if _, err := os.Stat(filepath.Join(dir, patchRunnerName)); err != nil {
		return "", fmt.Errorf("patch runner not found in plugin: %w", err)
	}
	return dir, nil
}

// PayloadFile is one file that will be executed as part of the patch
// payload, together with its SHA-256 fingerprint.
type PayloadFile struct {
	Path   string
	SHA256 string
}

// payloadScriptEnvVar mirrors run_patch.cmd: it names the private payload
// script beside the runner when set, instead of the single *.ps1 glob.
const payloadScriptEnvVar = "BN_PATCH_PAYLOAD"

func sha256File(path string) (string, error) {
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

// resolvePayloadScript finds the private payload script the runner will
// execute: $BN_PATCH_PAYLOAD beside the runner when set, otherwise the
// first *.ps1 in patchDir (filepath.Glob returns sorted order, matching
// the runner's "single *.ps1 beside this file" convention). Returns ""
// when no payload script is present.
func resolvePayloadScript(patchDir string) string {
	if name := strings.TrimSpace(os.Getenv(payloadScriptEnvVar)); name != "" {
		candidate := filepath.Join(patchDir, name)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
		return ""
	}
	matches, err := filepath.Glob(filepath.Join(patchDir, "*.ps1"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// PayloadFingerprint computes the SHA-256 hashes of every file that will
// run when the patch executes from patchDir: patch.bat, run_patch.cmd,
// and the resolved private payload script (*.ps1) when one is present.
// The payload is dropped-in, unaudited code that runs with Administrator
// rights, so its fingerprint is the identity the consent decision and
// the logs actually refer to. The hash of the private payload is only
// ever computed at runtime from the file on disk — it is not (and must
// not be) pinned in the source tree.
func PayloadFingerprint(patchDir string) ([]PayloadFile, error) {
	required := []string{patchScriptName, patchRunnerName}
	out := make([]PayloadFile, 0, len(required)+1)
	for _, name := range required {
		p := filepath.Join(patchDir, name)
		sum, err := sha256File(p)
		if err != nil {
			return nil, fmt.Errorf("fingerprint %s: %w", p, err)
		}
		out = append(out, PayloadFile{Path: p, SHA256: sum})
	}
	if payload := resolvePayloadScript(patchDir); payload != "" {
		sum, err := sha256File(payload)
		if err != nil {
			return nil, fmt.Errorf("fingerprint %s: %w", payload, err)
		}
		out = append(out, PayloadFile{Path: payload, SHA256: sum})
	}
	return out, nil
}

// logPayloadFingerprint records what is about to run, by hash, before
// any elevated execution. It fails closed: if the payload that is about
// to be executed cannot be fingerprinted, it must not run unnamed.
func logPayloadFingerprint(patchDir string) error {
	files, err := PayloadFingerprint(patchDir)
	if err != nil {
		return err
	}
	label := plugin.PatchPluginName
	if m, merr := plugin.ReadManifest(plugin.PatchPluginName); merr == nil {
		label = fmt.Sprintf("%s (id=%s, version=%s)", m.Name, m.ID, m.Version)
	}
	log.Printf("[patcher] payload fingerprint for plugin %s at %s:", label, patchDir)
	for _, f := range files {
		log.Printf("[patcher]   sha256=%s  path=%s", f.SHA256, f.Path)
	}
	return nil
}

func executePatchScriptWithAdmin(patchDir string) error {
	scriptPath := filepath.Join(patchDir, patchScriptName)

	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("patch script not found at %s: %w", scriptPath, err)
	}

	runnerPath := filepath.Join(patchDir, patchRunnerName)
	if _, err := os.Stat(runnerPath); err != nil {
		return fmt.Errorf("patch runner not found at %s: %w", runnerPath, err)
	}

	if err := logPayloadFingerprint(patchDir); err != nil {
		return fmt.Errorf("patch payload fingerprint failed: %w", err)
	}

	params := "/c " + patchScriptName + " & " + patchRunnerName
	log.Printf("[patcher] running %s after %s with administrator privileges", patchRunnerName, patchScriptName)

	verbPtr, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	filePtr, err := windows.UTF16PtrFromString("cmd.exe")
	if err != nil {
		return err
	}
	paramsPtr, err := windows.UTF16PtrFromString(params)
	if err != nil {
		return err
	}
	dirPtr, err := windows.UTF16PtrFromString(patchDir)
	if err != nil {
		return err
	}

	log.Printf("[patcher] Requesting Administrator privileges (UAC) for %s...", patchScriptName)

	r1, _, err := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		uintptr(unsafe.Pointer(paramsPtr)),
		uintptr(unsafe.Pointer(dirPtr)),
		windows.SW_HIDE,
	)

	if r1 <= 32 {
		log.Printf("[patcher] UAC elevation was rejected or failed: %v", err)
		return err
	}

	return nil
}

func isMinecraftPackageRegistered() bool {
	for _, pkg := range []string{"MICROSOFT.MINECRAFTUWP", "Microsoft.MinecraftWindowsBeta"} {
		if info, err := registry.GetAppxInfo(pkg); err == nil && info != nil {
			return true
		}
	}
	return false
}

func pickPatchRegisterVersion(mode string) string {
	vdir, err := apppath.VersionsDir()
	if err != nil || strings.TrimSpace(vdir) == "" {
		return ""
	}

	if mode == config.PatchRegisterLastLaunched {
		name := config.GetLastLaunchedVersion()
		if name != "" && utils.DirExists(filepath.Join(vdir, name)) {
			return name
		}
		log.Printf("[patcher] last-launched version %q not found, falling back to latest release", name)
	}

	metas, err := versions.ScanVersions(vdir)
	if err != nil || len(metas) == 0 {
		return ""
	}
	wantPreview := mode == config.PatchRegisterLatestPreview
	var best versions.VersionMeta
	found := false
	for _, m := range metas {
		isPreview := strings.EqualFold(strings.TrimSpace(m.Type), "preview")
		if isPreview != wantPreview {
			continue
		}
		if !found || m.CreatedAt.After(best.CreatedAt) ||
			(m.CreatedAt.Equal(best.CreatedAt) && m.Name > best.Name) {
			best, found = m, true
		}
	}
	if !found {
		return ""
	}
	return strings.TrimSpace(best.Name)
}

func ensureMinecraftRegistered() {
	if isMinecraftPackageRegistered() {
		return
	}

	mode := config.GetPatchRegisterMode()
	if mode == config.PatchRegisterOff {
		log.Printf("[patcher] auto-register disabled in settings, skipping version registration")
		return
	}

	name := pickPatchRegisterVersion(mode)
	if name == "" {
		log.Printf("[patcher] no suitable version found to auto-register (mode=%s)", mode)
		return
	}

	vdir, err := apppath.VersionsDir()
	if err != nil || strings.TrimSpace(vdir) == "" {
		log.Printf("[patcher] failed to resolve versions dir for auto-register: %v", err)
		return
	}

	log.Printf("[patcher] Minecraft package not registered; registering version %q (mode=%s)...", name, mode)
	if msg := gdk.RegisterVersionFolder(filepath.Join(vdir, name)); msg != "" {
		log.Printf("[patcher] auto-register failed for version %q: %s", name, msg)
		return
	}
	mcservice.ReconcileRegisteredFlags()
	log.Printf("[patcher] version %q registered successfully before patch", name)
}

func cleanupOldPatches() {
	tempDir := os.TempDir()
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, patchTempDirPrefix) {
			continue
		}

		fullPath := filepath.Join(tempDir, name)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > patchExtractionMaxAge {
			_ = os.RemoveAll(fullPath)
		}
	}
}

func RunAuto() {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("[patcher] panic in embedded patch execution: %v", recovered)
		}
	}()

	if plugin.IsAutoRunDeclined() {
		log.Printf("[patcher] auto-run skipped: elevation declined by user at startup")
		return
	}

	if config.GetAutoPatchDisabled() {
		log.Printf("[patcher] auto-run disabled in settings, skipping")
		return
	}

	cleanupOldPatches()

	patchDir, err := ResolveDir()
	if err != nil {
		log.Printf("[patcher] patch plugin not present, skipping auto-run: %v", err)
		return
	}

	ensureMinecraftRegistered()

	if err := executePatchScriptWithAdmin(patchDir); err != nil {
		log.Printf("[patcher] patch notice (non-blocking): %v", err)
		return
	}

	log.Printf("[patcher] patch executed successfully with Administrator rights.")
}

func RunManual() error {
	cleanupOldPatches()

	patchDir, err := ResolveDir()
	if err != nil {
		return fmt.Errorf("ERR_PATCH_PLUGIN_MISSING: %w", err)
	}

	ensureMinecraftRegistered()

	if err := executePatchScriptWithAdmin(patchDir); err != nil {
		return fmt.Errorf("ERR_PATCH_RUN_FAILED: %w", err)
	}

	log.Printf("[patcher] manual patch executed successfully with Administrator rights.")
	return nil
}

func RunAutoAsync() {
	go RunAuto()
}
