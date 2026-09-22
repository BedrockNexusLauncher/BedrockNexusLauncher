package launch

import (
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	quitRequested   atomic.Bool
	userHidLauncher atomic.Bool
)

// QuitRequested reports whether the current shutdown was asked for through
// QuitLauncher. The window-close hook relies on this to tell an explicit quit
// apart from the user pressing the close button: if it hid the window on an
// explicit quit, the launcher would stay alive in the tray forever.
func QuitRequested() bool {
	return quitRequested.Load()
}

// QuitLauncher terminates the launcher regardless of the minimise-to-tray
// setting. Every "quit" affordance outside the window close button goes
// through here.
func QuitLauncher() {
	quitRequested.Store(true)
	application.Get().Quit()
}

// GetMainWindow returns the launcher window, or nil while it does not exist
// yet: a monitor started from the single-instance pipe can run before the
// window is created.
func GetMainWindow() application.Window {
	w, ok := application.Get().Window.GetByName("main")
	if !ok {
		return nil
	}
	return w
}

// MarkLauncherHiddenByUser records that the user themselves sent the launcher
// to the tray. A finishing game must not drag it back out, which would override
// an explicit decision and cover whatever the user moved on to.
func MarkLauncherHiddenByUser() {
	userHidLauncher.Store(true)
}

// UserHidLauncher reports whether the user sent the launcher to the tray.
func UserHidLauncher() bool {
	return userHidLauncher.Load()
}

// RestoreLauncherWindow brings the launcher back to the foreground, keeping
// whatever size state it had. UnMinimise is the only restore step: Restore and
// the raw SW_RESTORE both drop a maximised window back to its pre-maximised
// size, so a maximised launcher would shrink on every game exit and tray click.
func RestoreLauncherWindow() {
	userHidLauncher.Store(false)

	w := GetMainWindow()
	if w == nil {
		return
	}
	w.Show()
	w.UnMinimise()
	w.Focus()
}
