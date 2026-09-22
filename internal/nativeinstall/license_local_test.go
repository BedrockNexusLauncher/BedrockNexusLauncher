package nativeinstall

import (
	"encoding/base64"
	"testing"
)

func fullLicenseResponse() contentLicenseResponse {
	raw := base64.StdEncoding.EncodeToString([]byte(`<License><LicenseInfo Type="Full"/></License>`))
	return contentLicenseResponse{
		License: &struct{ Keys []struct{ Value string } }{
			Keys: []struct{ Value string }{{Value: raw}},
		},
	}
}

func TestContentLicenseState(t *testing.T) {
	if got := contentLicenseState(contentLicenseResponse{}); got != "error" {
		t.Fatalf("expected error for empty response, got %q", got)
	}
	if got := contentLicenseState(contentLicenseResponse{
		SatisfactionFailure: &struct {
			Code        int64
			Description string
		}{Code: 1},
	}); got != "not_entitled" {
		t.Fatalf("expected not_entitled, got %q", got)
	}
	if got := contentLicenseState(fullLicenseResponse()); got != "authorized" {
		t.Fatalf("expected authorized, got %q", got)
	}
}

func TestValidContentID(t *testing.T) {
	if !validContentID("985D1EE4-0E9D-49DE-9A99-E208ADC08D0C") {
		t.Fatal("expected valid UUID-shaped content ID")
	}
	if validContentID("short") {
		t.Fatal("expected invalid for short ID")
	}
}
