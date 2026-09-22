package uwpdownload

import (
	"strings"
	"testing"
)

func TestValidChannelRejectsBogus(t *testing.T) {
	if ValidChannel("bogus-channel") {
		t.Fatal("expected bogus channel to be invalid")
	}
	for _, c := range []string{"release", "beta", "preview"} {
		if !ValidChannel(c) {
			t.Fatalf("expected %q to be valid", c)
		}
	}
}

func TestFilenameRejectsBadVersion(t *testing.T) {
	if _, err := Filename("not-a-version", "release"); err == nil {
		t.Fatal("expected error for bad version")
	}
	got, err := Filename("1.21.93.1", "release")
	if err != nil || got != "Minecraft-UWP-Release-1.21.93.1.appx" {
		t.Fatalf("unexpected filename %q err=%v", got, err)
	}
}

func TestParseCatalogRejectsGarbage(t *testing.T) {
	if _, err := ParseCatalog(strings.NewReader("garbage")); err == nil {
		t.Fatal("expected error for garbage catalog")
	}
}

func TestLoadCatalogHasEntries(t *testing.T) {
	entries, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected embedded catalog to have entries")
	}
}
