package uwp

import "testing"

func TestSafeRelativePathRejectsTraversal(t *testing.T) {
	if _, err := safeRelativePath(`..\evil.exe`); err == nil {
		t.Fatal("expected error for traversal path")
	}
	if _, err := safeRelativePath(`/abs/path`); err == nil {
		t.Fatal("expected error for absolute path")
	}
	got, err := safeRelativePath(`Minecraft.Windows.exe`)
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
