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
