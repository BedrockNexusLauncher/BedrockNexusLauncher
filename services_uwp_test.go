package main

import "testing"

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
