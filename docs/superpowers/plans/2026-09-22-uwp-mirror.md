# UWP Mirror Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port UWP install/launch support from LeviLauncher v1.0.x into Bedrock Nexus as an additive package-type.

**Architecture:** Mirror `internal/uwp/` + `internal/uwpdownload/` verbatim from upstream (package `uwp` unchanged), adapt only module path references; wire minimal RPC in `minecraft.go`; GDK and `patcher.go` untouched.

**Tech Stack:** Go 1.25, Wails v3, Windows-only deploy/activate (`deploy_windows.go`, `activate_windows.go`), stdlib `archive/zip`, `encoding/xml`.

**Spec:** `docs/superpowers/specs/2026-09-22-uwp-design.md` (and `specs/002-uwp-support/spec.md` FR-001..FR-009)

## Global Constraints

- Go floor is `1.25` per `go.mod`.
- Windows-only files keep `_windows.go` suffix and upstream build constraints verbatim.
- Package-type is additive: never unregister or break existing GDK instances.
- No admin elevation for UWP install/launch/uninstall paths.
- Stable error codes surfaced to UI: `ERR_UWP_MANIFEST`, `ERR_UWP_NOT_MINECRAFT`, `ERR_UWP_UNSAFE_ARCHIVE`, `ERR_UWP_ARCHIVE`, `ERR_UWP_ARCHITECTURE`, `ERR_UWP_DEPLOY`, `ERR_UWP_DEV_MODE`, `ERR_UWP_DEPENDENCY`, `ERR_UWP_CONFLICT`, `ERR_UWP_HASH`.
- GPL-3.0-only: copied files keep upstream license headers; update `THIRD_PARTY_NOTICES`.
- Upstream source of truth: `https://github.com/LiteLDev/LeviLauncher` `main` branch `internal/uwp/` (`manifest.go`, `extract.go`, `deploy_windows.go`, `activate_windows.go`, `full_trust.go`, `auth_key.go`) and `internal/uwpdownload/` (`catalog.go`, `resolve.go`).

---

### Task 1: Port safe extraction helpers

**Files:**
- Create: `internal/uwp/extract.go`
- Test: `internal/uwp/extract_test.go`

**Interfaces:**
- Consumes: stdlib only (`archive/zip`, `context`, `os`, `path/filepath`).
- Produces: `func safeRelativePath(name string) (string, error)`; `func architectureRank(candidate, target string) int`; `func Install(ctx context.Context, archivePath, targetDir string, opts Options) (Manifest, error)` — used by Task 2 (`ReadManifest` calls `safeRelativePath`) and Task 3 (deploy calls `Install`).

- [ ] **Step 1: Write the failing test**

```go
package uwp

import "testing"

func TestSafeRelativePathRejectsTraversal(t *testing.T) {
    if _, err := safeRelativePath(`..\evil.exe`); err == nil {
        t.Fatal("expected error for traversal path")
    }
    if _, err := safeRelativePath(`/abs/path`); err == nil {
        t.Fatal("expected error for absolute path")
    }
    got, err := safeRelativePath(`Minecraft.Windows.exe`);
    if err != nil || got == "" {
        t.Fatalf("expected clean path, got %q err=%v", got, err)
    }
}

func TestArchitectureRankPrefersX64(t *testing.T) {
    if architectureRank("x64", "amd64") != 0 {
        t.Fatal("expected x64 rank 0 on amd64")
    }
    if architectureRank("arm64", "amd64") == 0 {
        t.Fatal("arm64 must not rank 0 on amd64")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/uwp/ -run 'TestSafeRelativePathRejectsTraversal|TestArchitectureRankPrefersX64' -v`
Expected: FAIL with "no such file or directory" or "undefined: safeRelativePath"

- [ ] **Step 3: Write minimal implementation**

Copy verbatim from upstream (no module-path edits needed, stdlib only):
- Source: `https://raw.githubusercontent.com/LiteLDev/LeviLauncher/main/internal/uwp/extract.go`
- Dest: `internal/uwp/extract.go`
- Keep `maxExtractBytes = 20 << 30`, `Progress`, `Options`, `Install`, `extract`, `extractFile`, `selectBundlePackage` exactly as upstream.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/uwp/ -run 'TestSafeRelativePathRejectsTraversal|TestArchitectureRankPrefersX64' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/uwp/extract.go internal/uwp/extract_test.go
git commit -m "feat(uwp): port safe appx extraction helpers"
```

### Task 2: Port manifest validation

**Files:**
- Create: `internal/uwp/manifest.go`
- Test: `internal/uwp/manifest_test.go`

**Interfaces:**
- Consumes: `safeRelativePath` from Task 1.
- Produces: `type Manifest struct` with `func ReadManifest(dir string) (Manifest, error)`; `func (m Manifest) GameVersion() string`; `func (m Manifest) IsPreview() bool`; `func (m Manifest) FamilyName() string`; `func ErrorCode(err error) string` — used by Task 3 (deploy validates) and Task 6 (RPC returns `GameVersion`, `FamilyName`).

- [ ] **Step 1: Write the failing test**

```go
package uwp

import (
    "os"
    "path/filepath"
    "testing"
)

func TestReadManifestRejectsUnknownIdentity(t *testing.T) {
    dir := t.TempDir()
    bad := `<?xml version="1.0"?><Package><Identity Name="Fake.App" Publisher="CN=Microsoft Corporation, O=Microsoft Corporation, L=Redmond, S=Washington, C=US" Version="1.21.93.1" ProcessorArchitecture="x64"/><Applications><Application Id="App" Executable="Minecraft.Windows.exe"/></Applications></Package>`
    if err := os.WriteFile(filepath.Join(dir, "AppxManifest.xml"), []byte(bad), 0644); err != nil {
        t.Fatal(err)
    }
    if _, err := ReadManifest(dir); ErrorCode(err) != "ERR_UWP_NOT_MINECRAFT" {
        t.Fatalf("expected ERR_UWP_NOT_MINECRAFT, got %v", err)
    }
}

func TestGameVersionMaps1219301(t *testing.T) {
    m := Manifest{}
    m.Identity.Version = "1.21.9301.0"
    if got := m.GameVersion(); got != "1.21.93.1" {
        t.Fatalf("expected 1.21.93.1, got %q", got)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/uwp/ -run 'TestReadManifestRejectsUnknownIdentity|TestGameVersionMaps1219301' -v`
Expected: FAIL with "undefined: ReadManifest" or "undefined: Manifest"

- [ ] **Step 3: Write minimal implementation**

Copy verbatim from upstream:
- Source: `https://raw.githubusercontent.com/LiteLDev/LeviLauncher/main/internal/uwp/manifest.go`
- Dest: `internal/uwp/manifest.go`
- Keep `ReleasePackageName = "Microsoft.MinecraftUWP"`, `PreviewPackageName = "Microsoft.MinecraftWindowsBeta"`, `microsoftPublisher`, `Error` type with `Code` + `Cause` exactly.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/uwp/ -v`
Expected: PASS (all tests in package)

- [ ] **Step 5: Commit**

```bash
git add internal/uwp/manifest.go internal/uwp/manifest_test.go
git commit -m "feat(uwp): port manifest validation with stable error codes"
```

### Task 3: Port auth-key fix + Windows deploy

**Files:**
- Create: `internal/uwp/auth_key.go`
- Create: `internal/uwp/deploy_windows.go`
- Test: `internal/uwp/deploy_windows_test.go`

**Interfaces:**
- Consumes: `Manifest` + `Install` from Task 1–2; stdlib + `golang.org/x/sys/windows` (already in `go.mod`).
- Produces: `func EnsurePrereqs() error` returning `ERR_UWP_DEV_MODE` or `ERR_UWP_DEPENDENCY`; `func RegisterPackage(stagedDir string, manifest Manifest) error` returning `ERR_UWP_DEPLOY` / `ERR_UWP_CONFLICT` — used by Task 6 RPC `UwpInstall`.

- [ ] **Step 1: Write the failing test**

```go
package uwp

import "testing"

func TestErrorCodePassthrough(t *testing.T) {
    err := failure("ERR_UWP_DEV_MODE", nil)
    if ErrorCode(err) != "ERR_UWP_DEV_MODE" {
        t.Fatalf("expected ERR_UWP_DEV_MODE, got %q", ErrorCode(err))
    }
    if ErrorCode(nil) != "" {
        t.Fatal("expected empty code for nil")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/uwp/ -run TestErrorCodePassthrough -v`
Expected: PASS already (proves harness works) — then delete this file after deploy port; the real gate is `go vet` on windows files. If `deploy_windows.go` missing, next step adds it.

- [ ] **Step 3: Write minimal implementation**

Copy verbatim from upstream (keep `_windows.go` suffix and build constraints):
- Source: `https://raw.githubusercontent.com/LiteLDev/LeviLauncher/main/internal/uwp/auth_key.go` → `internal/uwp/auth_key.go`
- Source: `https://raw.githubusercontent.com/LiteLDev/LeviLauncher/main/internal/uwp/deploy_windows.go` → `internal/uwp/deploy_windows.go`
- Do not edit PowerShell encoded-command logic; only change import prefix if file references `github.com/LiteLDev/LeviLauncher/` to `github.com/BedrockNexusLauncher/BedrockNexusLauncher/`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go vet ./internal/uwp/`
Expected: PASS with no output
Run: `go test ./internal/uwp/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/uwp/auth_key.go internal/uwp/deploy_windows.go internal/uwp/deploy_windows_test.go
git commit -m "feat(uwp): port auth-key fix and dev-mode deploy"
```

### Task 4: Port activation + full-trust

**Files:**
- Create: `internal/uwp/activate_windows.go`
- Create: `internal/uwp/full_trust.go`
- Test: `internal/uwp/activate_windows_test.go`

**Interfaces:**
- Consumes: `Manifest.FamilyName()`, `Manifest.Applications[0].ID` from Task 2.
- Produces: `func Launch(familyName, appID string) error`; `func TrackExit(familyName string) error` (exact names taken from upstream file; if upstream names differ, use upstream names verbatim and record them here) — used by Task 6 RPC `UwpLaunch`.

- [ ] **Step 1: Write the failing test**

```go
package uwp

import "testing"

func TestFamilyNamesMatchStoreIdentities(t *testing.T) {
    m := Manifest{}
    m.Identity.Name = ReleasePackageName
    if m.FamilyName() != ReleaseFamilyName {
        t.Fatalf("release family mismatch: %q", m.FamilyName())
    }
    m.Identity.Name = PreviewPackageName
    if m.FamilyName() != PreviewFamilyName {
        t.Fatalf("preview family mismatch: %q", m.FamilyName())
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/uwp/ -run TestFamilyNamesMatchStoreIdentities -v`
Expected: PASS (guards constants before windows-only code lands)

- [ ] **Step 3: Write minimal implementation**

Copy verbatim from upstream:
- Source: `https://raw.githubusercontent.com/LiteLDev/LeviLauncher/main/internal/uwp/activate_windows.go` → `internal/uwp/activate_windows.go`
- Source: `https://raw.githubusercontent.com/LiteLDev/LeviLauncher/main/internal/uwp/full_trust.go` → `internal/uwp/full_trust.go`

- [ ] **Step 4: Run test to verify it passes**

Run: `go vet ./internal/uwp/`
Expected: PASS
Run: `go test ./internal/uwp/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/uwp/activate_windows.go internal/uwp/full_trust.go internal/uwp/activate_windows_test.go
git commit -m "feat(uwp): port activation and full-trust launch"
```

### Task 5: Port Windows Update catalogue (uwpdownload)

**Files:**
- Create: `internal/uwpdownload/catalog.go`
- Create: `internal/uwpdownload/resolve.go`
- Test: `internal/uwpdownload/catalog_test.go`

**Interfaces:**
- Consumes: existing `internal/downloader.Manager` created by `downloader.NewManager(events Events, opts Options)` (see `internal/downloader/downloader.go:74`).
- Produces: `func ListVersions(channel string) ([]Version, error)` where `type Version struct { Version string; UpdateID string; Channel string }`; `func ResolveDownloadURL(updateID string) (url string, sha256 string, err error)` — used by Task 6 RPC `UwpListVersions` / `UwpInstall`.

- [ ] **Step 1: Write the failing test**

```go
package uwpdownload

import "testing"

func TestListVersionsRejectsUnknownChannel(t *testing.T) {
    if _, err := ListVersions("bogus-channel"); err == nil {
        t.Fatal("expected error for unknown channel")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/uwpdownload/ -run TestListVersionsRejectsUnknownChannel -v`
Expected: FAIL with "undefined: ListVersions"

- [ ] **Step 3: Write minimal implementation**

Copy verbatim from upstream:
- Source: `https://github.com/LiteLDev/LeviLauncher/tree/main/internal/uwpdownload` (`catalog.go`, `resolve.go`, plus `versions.json` handling as upstream does)
- Dest: `internal/uwpdownload/catalog.go`, `internal/uwpdownload/resolve.go`
- Change only import prefix `github.com/LiteLDev/LeviLauncher` → `github.com/BedrockNexusLauncher/BedrockNexusLauncher` if present.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/uwpdownload/ -v`
Expected: PASS
Run: `go vet ./internal/uwpdownload/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/uwpdownload/catalog.go internal/uwpdownload/resolve.go internal/uwpdownload/catalog_test.go
git commit -m "feat(uwp): port Windows Update catalogue resolve"
```

### Task 6: Wire RPC + instance badge

**Files:**
- Modify: `minecraft.go:1-60` (add imports + methods on existing service struct)
- Modify: `frontend/src/pages/InstanceSelectPage.tsx` (badge + filter)
- Test: `internal/uwp/rpc_smoke_test.go` (documents RPC error-code mapping; runs without Store)

**Interfaces:**
- Consumes: `uwp.ReadManifest`, `uwp.Install`, `uwp.RegisterPackage`, `uwp.Launch`, `uwpdownload.ListVersions`, `uwpdownload.ResolveDownloadURL` from Tasks 1–5.
- Produces: `func (m *Minecraft) UwpListVersions(channel string) ([]UwpVersion, error)`; `func (m *Minecraft) UwpGetPrereqs() (UwpPrereqs, error)`; `func (m *Minecraft) UwpInstall(updateID string) (string, error)`; `func (m *Minecraft) UwpLaunch(instanceDir string) error` — called from frontend via Wails bindings.

- [ ] **Step 1: Write the failing test**

```go
package main

import "testing"

func TestUwpErrorCodesAreStable(t *testing.T) {
    codes := map[string]bool{
        "ERR_UWP_DEV_MODE": true, "ERR_UWP_DEPENDENCY": true,
        "ERR_UWP_CONFLICT": true, "ERR_UWP_HASH": true,
    };
    _ = codes
}
```

(This compiles immediately; the real failing gate is `go vet` complaining that `UwpListVersions` is undefined — verified in Step 2.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go vet ./... 2>&1 | Select-String "UwpListVersions"`
Expected: FAIL finding undefined method (proves wiring missing)

- [ ] **Step 3: Write minimal implementation**

In `minecraft.go` add imports:

```go
"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/uwp"
"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/uwpdownload"
```

Add types + methods (return upstream `ErrorCode(err)` strings verbatim, never raw PowerShell):

```go
type UwpVersion struct {
    Version string `json:"version"`
    UpdateID string `json:"updateId"`
    Channel string `json:"channel"`
}

type UwpPrereqs struct {
    DeveloperMode bool `json:"developerMode"`
    MissingDeps []string `json:"missingDeps"`
}
```

`UwpListVersions` calls `uwpdownload.ListVersions`; `UwpGetPrereqs` calls `uwp.EnsurePrereqs` and maps to struct; `UwpInstall` resolves URL via `uwpdownload.ResolveDownloadURL`, downloads with existing `downloader.NewManager`, verifies SHA-256, calls `uwp.Install` then `uwp.RegisterPackage`; `UwpLaunch` reads manifest via `uwp.ReadManifest` then calls `uwp.Launch`.

In `frontend/src/pages/InstanceSelectPage.tsx` add badge text `UWP` vs `GDK` next to instance name and a filter dropdown with values `all`, `gdk`, `uwp` (no styling changes beyond existing `componentStyles`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go vet ./internal/uwp/ ./internal/uwpdownload/ .`
Expected: PASS
Run: `go test ./internal/uwp/ ./internal/uwpdownload/ -v`
Expected: PASS
Run: `npm --prefix frontend run build`
Expected: PASS (proves badge compiles)

- [ ] **Step 5: Commit**

```bash
git add minecraft.go frontend/src/pages/InstanceSelectPage.tsx internal/uwp/rpc_smoke_test.go
git commit -m "feat(uwp): wire install/launch RPC and instance badge"
```

### Task 7: License + map + verify

**Files:**
- Modify: `THIRD_PARTY_NOTICES`
- Modify: `PROJECT_MAP.md`
- Test: full `go test ./...` (windows) — no new test file, verification only.

**Interfaces:**
- Consumes: all Tasks 1–6.
- Produces: releasable UWP phase with docs updated.

- [ ] **Step 1: Write the failing test**

Run: `Select-String -Path THIRD_PARTY_NOTICES -Pattern "LeviLauncher" -SimpleMatch`
Expected: FAIL (no match yet — proves attribution missing)

- [ ] **Step 2: Run test to verify it fails**

Same command as Step 1; confirm no output.

- [ ] **Step 3: Write minimal implementation**

Append to `THIRD_PARTY_NOTICES`:

```
LeviLauncher (github.com/LiteLDev/LeviLauncher) — GPL-3.0-only.
UWP support (internal/uwp, internal/uwpdownload) mirrored from LeviLauncher v1.0.x with module-path adaptation.
```

In `PROJECT_MAP.md` under `/internal` table add rows:

```
| **uwp** | manifest/extract/deploy/activate/auth-key | UWP install + launch, additive to GDK |
| **uwpdownload** | catalog/resolve | Windows Update catalogue → CDN URL + digest |
```

- [ ] **Step 4: Run test to verify it passes**

Run: `Select-String -Path THIRD_PARTY_NOTICES -Pattern "LeviLauncher" -SimpleMatch`
Expected: PASS (match found)
Run: `go test ./internal/uwp/ ./internal/uwpdownload/ -v`
Expected: PASS
Run: `go vet ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add THIRD_PARTY_NOTICES PROJECT_MAP.md
git commit -m "docs: attribute LeviLauncher UWP sources and update project map"
```
