// Package plugin — elevation consent state for the personal patch plugin.
//
// The "run with admin / continue without admin" consent flow belongs to the
// patch system, so its decision state lives here and not in the launcher core.
// main.go / minecraft.go only call into these helpers and perform the OS-level
// relaunch (which must stay in package main: it needs os.Executable and the
// single-instance guard). Gating rule: the consent page is shown only when the
// patch plugin is present; when it is absent the launcher asks nothing and
// goes straight home.
package plugin

import "sync/atomic"

var (
	// consentPending is true while the in-app elevation consent page
	// (ElevationConsentPage) is still unanswered; auto-patch waits for it.
	consentPending atomic.Bool

	// autoRunDeclined is true when the user declined elevation at startup;
	// auto-patch is then skipped for the rest of the session.
	autoRunDeclined atomic.Bool
)

// ShouldAskElevation reports whether the launcher should show the elevation
// consent page. True only when the patch plugin is installed, auto-patch is
// enabled in settings, and the process is not already elevated.
func ShouldAskElevation(autoPatchDisabled bool, elevated bool) bool {
	if autoPatchDisabled {
		return false
	}
	if elevated {
		return false
	}
	return IsPatchPresent()
}

// SetConsentPending marks the consent page as pending (shown, unanswered).
func SetConsentPending(v bool) { consentPending.Store(v) }

// ClearConsentPending clears the pending flag after accept or decline.
func ClearConsentPending() { consentPending.Store(false) }

// IsConsentPending reports whether consent is still unanswered.
func IsConsentPending() bool { return consentPending.Load() }

// SetAutoRunDeclined records that the user declined elevation at startup.
func SetAutoRunDeclined(v bool) { autoRunDeclined.Store(v) }

// IsAutoRunDeclined reports whether auto-patch must be skipped this session.
func IsAutoRunDeclined() bool { return autoRunDeclined.Load() }
