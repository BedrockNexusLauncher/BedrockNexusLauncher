package gameinput

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/apppath"
	"github.com/wailsapp/wails/v3/pkg/application"
	winreg "golang.org/x/sys/windows/registry"
)

var (
	mu       sync.Mutex
	ensuring bool
)

// keyExists reports whether the given registry key can be opened. The
// handle is closed on every path — the OpenKey results were previously
// discarded, leaking a handle per call.
func keyExists(path string) bool {
	key, err := winreg.OpenKey(winreg.LOCAL_MACHINE, path, winreg.READ)
	if err != nil {
		return false
	}
	_ = key.Close()
	return true
}

func IsInstalled() bool {
	return keyExists(`SOFTWARE\\Microsoft\\GameInputRedist`) || keyExists(`SOFTWARE\\Microsoft\\GameInput`)
}

func EnsureInteractive(ctx context.Context) {
	mu.Lock()
	if ensuring {
		mu.Unlock()
		return
	}
	ensuring = true
	mu.Unlock()
	defer func() {
		mu.Lock()
		ensuring = false
		mu.Unlock()
	}()

	if IsInstalled() {
		return
	}

	application.Get().Event.Emit("gameinput.ensure.start", struct{}{})

	if len(msiBytes) == 0 {
		application.Get().Event.Emit("gameinput.download.error", "MSI not embedded")
		log.Println("gameinput msi not embedded")
		application.Get().Event.Emit("gameinput.ensure.done", false)
		return
	}

	dir, _ := apppath.InstallersDir()
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		application.Get().Event.Emit("gameinput.download.error", err.Error())
		log.Println("mkdir installers error:", err)
		application.Get().Event.Emit("gameinput.ensure.done", false)
		return
	}
	dlPath := filepath.Join(dir, "GameInputRedist.msi")
	tmpPath := dlPath + ".part"

	application.Get().Event.Emit("gameinput.download.start", int64(len(msiBytes)))
	if err := os.WriteFile(tmpPath, msiBytes, 0644); err != nil {
		application.Get().Event.Emit("gameinput.download.error", err.Error())
		log.Println("gameinput write error:", err)
		application.Get().Event.Emit("gameinput.ensure.done", false)
		return
	}
	if _, stErr := os.Stat(dlPath); stErr == nil {
		_ = os.Remove(dlPath)
	}
	if err := os.Rename(tmpPath, dlPath); err != nil {
		log.Println("gameinput rename error:", err)
		application.Get().Event.Emit("gameinput.download.error", err.Error())
		application.Get().Event.Emit("gameinput.ensure.done", false)
		return
	}
	application.Get().Event.Emit("gameinput.download.done", struct{}{})
	log.Println("GameInputRedist prepared:", dlPath)

	cmd := exec.Command("msiexec", "/i", dlPath, "/passive")
	if err := cmd.Run(); err != nil {
		log.Println("failed to run GameInput installer:", err)
	}
	installed := false
	for i := 0; i < 30; i++ {
		if keyExists(`SOFTWARE\\Microsoft\\GameInput`) || keyExists(`SOFTWARE\\Microsoft\\GameInputRedist`) {
			installed = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	log.Println("GameInput installed:", installed)
	application.Get().Event.Emit("gameinput.ensure.done", installed)
}
