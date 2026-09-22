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
