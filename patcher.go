package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/apppath"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/config"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/gdk"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/mcservice"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/registry"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/utils"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/versions"
	"golang.org/x/sys/windows"
)

//go:embed all:Patch
var patcherAssets embed.FS

const (
	patchTempDirPrefix    = "BedrockNexus_Patch_"
	patchScriptName       = "patch.bat"
	patchRunnerName       = "run_patch.cmd"
	patchExtractionMaxAge = 24 * time.Hour
)

var (
	shell32           = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")

	// patchAutoRunDeclined وقتی true باشد، کاربر درخواست ارتقای دسترسی هنگام استارت را رد کرده
	// و اجرای خودکار پچ در همین نشست نادیده گرفته می‌شود.
	patchAutoRunDeclined atomic.Bool

	// elevationConsentPending وقتی true باشد، درخواست ارتقای دسترسی هنوز در رابط گرافیکی
	// (صفحه ElevationConsentPage) بی‌پاسخ است و اجرای خودکار پچ تا تصمیم کاربر به تعویق می‌افتد.
	elevationConsentPending atomic.Bool
)

// SetPatchAutoRunDeclined وضعیت «رد کردن ارتقای دسترسی» را ثبت می‌کند
func SetPatchAutoRunDeclined() { patchAutoRunDeclined.Store(true) }

// extractEmbeddedPatch پوشه امبد شده را در دایرکتوری موقت استخراج می‌کند
func extractEmbeddedPatch() (string, error) {
	tempDir := os.TempDir()
	patchDir := filepath.Join(tempDir, patchTempDirPrefix+fmt.Sprintf("%d", time.Now().UnixNano()))

	if err := os.MkdirAll(patchDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create patch temp directory: %w", err)
	}

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel("Patch", path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(patchDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := fs.ReadFile(patcherAssets, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		if err := os.WriteFile(targetPath, data, 0755); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		return nil
	}

	if err := fs.WalkDir(patcherAssets, "Patch", walkFn); err != nil {
		_ = os.RemoveAll(patchDir)
		return "", fmt.Errorf("failed to extract embedded patch: %w", err)
	}

	return patchDir, nil
}

// executePatchScriptWithAdmin اسکریپت‌های پچ را با دسترسی ادمین (UAC) ویندوز اجرا می‌کند.
// ابتدا patch.bat و سپس wrapper قابل‌اعتماد run_patch.cmd را در همان cmd.exe اجرا
// می‌کند تا نصب GDK از مسیر wrapper انجام شود و فقط یک بار UAC نمایش داده شود.
func executePatchScriptWithAdmin(patchDir string) error {
	scriptPath := filepath.Join(patchDir, patchScriptName)

	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("patch script not found at %s: %w", scriptPath, err)
	}

	runnerPath := filepath.Join(patchDir, patchRunnerName)
	if _, err := os.Stat(runnerPath); err != nil {
		return fmt.Errorf("patch runner not found at %s: %w", runnerPath, err)
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

	// اجرای اسکریپت در حالت ادمین با پنجره کاملا مخفی (بدون نمایش ترمینال)
	r1, _, err := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		uintptr(unsafe.Pointer(paramsPtr)),
		uintptr(unsafe.Pointer(dirPtr)),
		windows.SW_HIDE,
	)

	// در توابع ویندوز مقادیر بزرگتر از ۳۲ به معنی موفقیت است
	if r1 <= 32 {
		log.Printf("[patcher] UAC elevation was rejected or failed: %v", err)
		return err
	}

	return nil
}

// isMinecraftPackageRegistered بررسی می‌کند پکیج ماینکرفت (نصب Store/Xbox App یا
// نسخهٔ ثبت‌شده توسط لانچر) از قبل در سیستم موجود باشد
func isMinecraftPackageRegistered() bool {
	for _, pkg := range []string{"MICROSOFT.MINECRAFTUWP", "Microsoft.MinecraftWindowsBeta"} {
		if info, err := registry.GetAppxInfo(pkg); err == nil && info != nil {
			return true
		}
	}
	return false
}

// pickPatchRegisterVersion بر اساس حالت انتخابی کاربر، نام نسخهٔ کاندید برای
// ثبت خودکار را برمی‌گرداند (جدیدترین نسخهٔ Release/Preview یا آخرین نسخهٔ اجرا شده)
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

// ensureMinecraftRegistered قبل از اجرای اسکریپت‌های پچ، اگر پکیج ماینکرفت در سیستم
// ثبت نشده باشد، بر اساس تنظیمات کاربر یک نسخه را ثبت می‌کند (payload خصوصی برای پیدا
// کردن پوشهٔ Content بازی به پکیج ثبت‌شده یا نصب Xbox App نیاز دارد). ثبت AppX
// per-user است و به دسترسی ادمین نیازی ندارد.
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

// cleanupOldPatches پاکسازی پوشه‌های پچ قدیمی
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

// runEmbeddedPatch اجرای فرآیند کلی پچ (فقط برای اجرای خودکار هنگام استارت)
func runEmbeddedPatch() {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("[patcher] panic in embedded patch execution: %v", recovered)
		}
	}()

	// اگر کاربر ارتقای دسترسی هنگام استارت را رد کرده باشد، پچ خودکار اجرا نمی‌شود
	if patchAutoRunDeclined.Load() {
		log.Printf("[patcher] auto-run skipped: elevation declined by user at startup")
		return
	}

	// اگر گزینه «اجرای خودکار هنگام استارت» در تنظیمات خاموش باشد، اجرا نمی‌شود
	if config.GetAutoPatchDisabled() {
		log.Printf("[patcher] auto-run disabled in settings, skipping")
		return
	}

	cleanupOldPatches()

	patchDir, err := extractEmbeddedPatch()
	if err != nil {
		log.Printf("[patcher] failed to extract embedded patch: %v", err)
		return
	}

	// قبل از اجرای اسکریپت‌ها، اگر پکیج ماینکرفت ثبت نباشد طبق تنظیمات کاربر
	// یک نسخه به‌صورت خودکار ثبت می‌شود (پیش‌نیاز payload خصوصی)
	ensureMinecraftRegistered()

	if err := executePatchScriptWithAdmin(patchDir); err != nil {
		log.Printf("[patcher] patch notice (non-blocking): %v", err)
		return
	}

	log.Printf("[patcher] patch executed successfully with Administrator rights.")
}

// RunPatchScriptManual اجرای دستی اسکریپت پچ به درخواست کاربر از رابط گرافیکی
func RunPatchScriptManual() error {
	cleanupOldPatches()

	patchDir, err := extractEmbeddedPatch()
	if err != nil {
		return fmt.Errorf("ERR_PATCH_EXTRACT_FAILED: %w", err)
	}

	// مثل اجرای خودکار، قبل از پچ ثبت خودکار نسخه انجام می‌شود
	ensureMinecraftRegistered()

	if err := executePatchScriptWithAdmin(patchDir); err != nil {
		return fmt.Errorf("ERR_PATCH_RUN_FAILED: %w", err)
	}

	log.Printf("[patcher] manual patch executed successfully with Administrator rights.")
	return nil
}

// runEmbeddedPatchAsync اجرای ناهمگام در پس‌زمینه
func runEmbeddedPatchAsync() {
	go runEmbeddedPatch()
}