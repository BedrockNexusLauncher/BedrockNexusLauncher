package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUwpListVersionsFiltersChannel(t *testing.T) {
	s := &VersionService{}
	if got := s.UwpListVersions("bogus-channel"); len(got) != 0 {
		t.Fatalf("expected empty for bogus channel, got %d", len(got))
	}
	release := s.UwpListVersions("release")
	if len(release) == 0 {
		t.Fatal("expected embedded catalog to have release entries")
	}
	for _, v := range release {
		if v.Type != "release" {
			t.Fatalf("expected only release entries, got %q", v.Type)
		}
	}
}

func TestUwpGetPrereqsShape(t *testing.T) {
	s := &VersionService{}
	p := s.UwpGetPrereqs()
	_ = p.DeveloperMode
}

func TestUwpInstanceDirTrims(t *testing.T) {
	if uwpInstanceDir(" padded ") != uwpInstanceDir("padded") {
		t.Fatalf("padded name must resolve to trimmed dir: %q", uwpInstanceDir(" padded "))
	}
	if filepath.Base(uwpInstanceDir("padded")) != "padded" {
		t.Fatalf("unexpected base: %q", uwpInstanceDir("padded"))
	}
}

func TestDownloadURLRejectsNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := downloadURL(context.Background(), srv.URL, dest); err == nil {
		t.Fatal("expected error for 403 response")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("expected no file created for 403 response")
	}
}
