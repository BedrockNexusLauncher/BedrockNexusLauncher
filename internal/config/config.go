package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	json "github.com/goccy/go-json"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/apppath"
)

var (
	cachedConfig AppConfig
	configMutex  sync.RWMutex
	isLoaded     bool
)

type AppConfig struct {
	BaseRoot          string `json:"base_root"`
	WindowWidth       int    `json:"window_width"`
	WindowHeight      int    `json:"window_height"`
	DisableDiscordRPC bool   `json:"disable_discord_rpc"`
	EnableBetaUpdates bool   `json:"enable_beta_updates"`
	// DisableAutoPatch اجرای خودکار اسکریپت پچ هنگام استارت لانچر را غیرفعال می‌کند
	// (پیش‌فرض false یعنی اجرای خودکار فعال است تا رفتار قبلی حفظ شود)
	DisableAutoPatch bool `json:"disable_auto_patch"`
	// PatchRegisterMode تعیین می‌کند قبل از اجرای پچ، در صورت ثبت نبودن پکیج
	// ماینکرفت در سیستم، کدام نسخه به‌صورت خودکار ثبت شود.
	// مقادیر مجاز: latest_release (پیش‌فرض) | latest_preview | last_launched | off
	PatchRegisterMode string `json:"patch_register_mode"`
	// LastLaunchedVersion آخرین نسخه‌ای که با لانچر اجرا شده است؛ برای حالت
	// last_launched مربوط به PatchRegisterMode استفاده می‌شود
	LastLaunchedVersion string `json:"last_launched_version"`
}

// Patch register mode values (see AppConfig.PatchRegisterMode)
const (
	PatchRegisterLatestRelease = "latest_release"
	PatchRegisterLatestPreview = "latest_preview"
	PatchRegisterLastLaunched  = "last_launched"
	PatchRegisterOff           = "off"
)

// NormalizePatchRegisterMode مقدار نامعتبر یا خالی را به پیش‌فرض (latest_release) برمی‌گرداند
func NormalizePatchRegisterMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case PatchRegisterLatestPreview:
		return PatchRegisterLatestPreview
	case PatchRegisterLastLaunched:
		return PatchRegisterLastLaunched
	case PatchRegisterOff:
		return PatchRegisterOff
	default:
		return PatchRegisterLatestRelease
	}
}

func Load() (AppConfig, error) {
	configMutex.RLock()
	if isLoaded {
		c := cachedConfig
		configMutex.RUnlock()
		return c, nil
	}
	configMutex.RUnlock()

	return Reload()
}

func Reload() (AppConfig, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	var c AppConfig
	p := apppath.ConfigPath()
	if b, err := os.ReadFile(p); err == nil {
		if err := json.Unmarshal(b, &c); err != nil {
			log.Printf("config.Reload: invalid config at %s: %v", p, err)
			return AppConfig{}, fmt.Errorf("ERR_CONFIG_CORRUPTED: %w", err)
		}
		cachedConfig = c
		isLoaded = true
		apppath.SetBaseRootOverride(c.BaseRoot)
		return c, nil
	} else if !os.IsNotExist(err) {
		log.Printf("config.Reload: read config failed at %s: %v", p, err)
		return AppConfig{}, fmt.Errorf("ERR_CONFIG_READ_FAILED: %w", err)
	}

	db, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log.Printf("config.Reload: marshal default config failed: %v", err)
		return AppConfig{}, err
	}
	if err := os.WriteFile(p, db, 0o644); err != nil {
		log.Printf("config.Reload: write default config failed at %s: %v", p, err)
		return c, fmt.Errorf("ERR_CONFIG_WRITE_FAILED: %w", err)
	}
	cachedConfig = c
	isLoaded = true
	apppath.SetBaseRootOverride(c.BaseRoot)
	return c, nil
}

func Save(c AppConfig) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	p := apppath.ConfigPath()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	cachedConfig = c
	isLoaded = true
	apppath.SetBaseRootOverride(c.BaseRoot)
	return os.WriteFile(p, b, 0o644)
}

func ConfigDir() string {
	return apppath.ConfigDir()
}

func GetBaseRootOverride() string {
	configMutex.RLock()
	if isLoaded {
		v := cachedConfig.BaseRoot
		configMutex.RUnlock()
		return strings.TrimSpace(v)
	}
	configMutex.RUnlock()

	c, _ := Load()
	return strings.TrimSpace(c.BaseRoot)
}

func GetDiscordRPCDisabled() bool {
	configMutex.RLock()
	if isLoaded {
		v := cachedConfig.DisableDiscordRPC
		configMutex.RUnlock()
		return v
	}
	configMutex.RUnlock()

	c, _ := Load()
	return c.DisableDiscordRPC
}

func GetAutoPatchDisabled() bool {
	configMutex.RLock()
	if isLoaded {
		v := cachedConfig.DisableAutoPatch
		configMutex.RUnlock()
		return v
	}
	configMutex.RUnlock()

	c, _ := Load()
	return c.DisableAutoPatch
}

// GetPatchRegisterMode حالت انتخاب نسخه برای ثبت خودکار قبل از پچ را برمی‌گرداند
// (مقدار خالی/نامعتبر به پیش‌فرض latest_release نرمالایز می‌شود)
func GetPatchRegisterMode() string {
	configMutex.RLock()
	if isLoaded {
		v := NormalizePatchRegisterMode(cachedConfig.PatchRegisterMode)
		configMutex.RUnlock()
		return v
	}
	configMutex.RUnlock()

	c, _ := Load()
	return NormalizePatchRegisterMode(c.PatchRegisterMode)
}

// GetLastLaunchedVersion آخرین نسخه اجرا شده توسط لانچر را برمی‌گرداند
func GetLastLaunchedVersion() string {
	configMutex.RLock()
	if isLoaded {
		v := strings.TrimSpace(cachedConfig.LastLaunchedVersion)
		configMutex.RUnlock()
		return v
	}
	configMutex.RUnlock()

	c, _ := Load()
	return strings.TrimSpace(c.LastLaunchedVersion)
}

// SetLastLaunchedVersion آخرین نسخه اجرا شده را ذخیره می‌کند (خطا فقط لاگ می‌شود؛
// ثبت نشدن آن نباید مسیر اجرای بازی را مختل کند)
func SetLastLaunchedVersion(name string) {
	c, err := Load()
	if err != nil {
		log.Printf("config.SetLastLaunchedVersion: load failed: %v", err)
		return
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == c.LastLaunchedVersion {
		return
	}
	c.LastLaunchedVersion = trimmed
	if err := Save(c); err != nil {
		log.Printf("config.SetLastLaunchedVersion: save failed: %v", err)
	}
}
