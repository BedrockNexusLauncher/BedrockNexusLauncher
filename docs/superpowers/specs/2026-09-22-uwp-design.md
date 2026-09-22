# UWP Support Design — Bedrock Nexus (mirror upstream v1.0.x)

Date: 2026-09-22
Scope: UWP first (from LeviLauncher v0.3.13 → v1.0.3 sync, approach A)
Status: Approved in chat (architecture + install flow + errors/tests)

## 1. Architecture
- New `internal/uwp/` mirroring upstream: `manifest.go`, `extract.go`, `deploy_windows.go`, `activate_windows.go`, `full_trust.go`, `auth_key.go` (+ tests).
- New `internal/uwpdownload/` for Windows Update catalogue: `catalog.go`, `resolve.go`, `versions.json` handling, digest verify.
- Additive package-type: existing `internal/gdk/`, `patcher.go`, GDK flows untouched. Instance list shows `GDK` / `UWP` badge + filter.
- Module path adapted from `github.com/LiteLDev/LeviLauncher` to `github.com/BedrockNexusLauncher/BedrockNexusLauncher`.
- License: GPL-3.0-only headers + `THIRD_PARTY_NOTICES` update (upstream is GPL-3.0-only).

## 2. Components
- Manifest validation: package identity = Minecraft publisher, allowed executable names, arch (x64/x86/ARM64), min OS version.
- Safe extraction: appx/zip-bundle with traversal guards, temp dir cleanup.
- Deploy: PowerShell encoded-command, per-user dev-mode registration, preserve GDK registrations, byte-for-byte restore of touched manifests.
- Activate: COM `IApplicationActivationManager` via AUMID, track lifetime until exit.
- Catalogue: Windows Update update-IDs → expiring CDN URLs at download time, hash-verified; keep newly discovered versions across restarts until in historical catalog (v1.0.2 behavior).

## 3. Data flow
- Install: browse catalogue (Release `Microsoft.MinecraftUWP` / Preview `Microsoft.MinecraftWindowsBeta`) → prereq check (Dev Mode, frameworks, VC++) → download via existing `internal/downloader` (resumable) → digest verify → manifest validate → register → create instance with isolated LocalState-backed data dir.
- Launch: Launch button → auth-key check (v1.0.2: save per-UWP-instance, update outdated keys) → activate → track running → on exit return to launcher.
- Coexistence: GDK + UWP Release + UWP Preview side by side, separate data dirs.

## 4. Error handling
- Stable localizable codes (no raw PowerShell to UI):
  - `ERR_UWP_DEV_MODE`, `ERR_UWP_DEPENDENCY`, `ERR_UWP_CONFLICT`, `ERR_UWP_HASH`, `ERR_UWP_MANIFEST`, `ERR_UWP_MIN_OS`
- Diagnostics attached for bug reports via `Minecraft.GetDiagnosticsReport()`.
- x86/ARM64: vanilla launch only (no native loader, matches upstream). x64: console toggle + `preload-native` supported.

## 5. Testing
- `go test ./internal/uwp/... ./internal/uwpdownload/...` (manifest, extract traversal, deploy args, activation params mocked on Windows).
- Manual matrix: install UWP Release, launch to menu, install GDK alongside, toggle Dev Mode off (expect guided block), corrupt download (expect hash error).
- No admin elevation for install/launch/uninstall paths.

## Out of scope (this phase)
- Xbox sign-in, tray icon, material personalization, Bedrinth rename, RenderDragon removal — next phases after UWP lands.
- Auto LeviLamina for UWP, Store license provisioning, .NET dependency.

## Self-review
- No TBDs. Architecture matches FR-001..FR-009 in `specs/002-uwp-support/spec.md`. Scope is single phase (UWP only), not whole v1.0.3.
