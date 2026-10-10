package mcservice

import "testing"

func TestFetchUWPVersionsLoadsCatalog(t *testing.T) {
	versions, err := FetchUWPVersions()
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) == 0 {
		t.Fatal("expected embedded UWP catalog to have entries")
	}
}

func TestResolveDownloadedUWPMissing(t *testing.T) {
	if got := ResolveDownloadedUWP("9.9.9.9", "release"); got != "" {
		t.Fatalf("expected empty for missing download, got %q", got)
	}
}

func TestDeleteDownloadedUWPMissing(t *testing.T) {
	if code := DeleteDownloadedUWP("9.9.9.9", "release"); code != "ERR_UWP_PACKAGE_NOT_FOUND" {
		t.Fatalf("expected ERR_UWP_PACKAGE_NOT_FOUND, got %q", code)
	}
}

func TestDeleteDownloadedUWPBadChannel(t *testing.T) {
	if code := DeleteDownloadedUWP("1.21.93.1", "bogus"); code != "ERR_UWP_PACKAGE_NOT_FOUND" {
		t.Fatalf("expected ERR_UWP_PACKAGE_NOT_FOUND, got %q", code)
	}
}

func TestGetVersionStatusMissingUWP(t *testing.T) {
	st := GetVersionStatus("9.9.9.9", "release")
	if st.IsDownloaded {
		t.Fatalf("expected IsDownloaded=false for missing UWP download")
	}
	// Channel matching is case-insensitive; a bogus channel must not report downloaded.
	st = GetVersionStatus("9.9.9.9", "Release")
	if st.IsDownloaded {
		t.Fatalf("expected IsDownloaded=false for missing UWP download with capitalized channel")
	}
}
