//go:build !windows

package httpx

import "net/http"

// newBaseTransport mirrors the Windows implementation in
// proxy_windows.go so this package also compiles off Windows (the
// launcher itself is Windows-only; this exists for tooling and tests
// on other platforms). There is no Windows system proxy to consult
// here, so the plain cloned default transport is used.
func newBaseTransport() *http.Transport {
	return http.DefaultTransport.(*http.Transport).Clone()
}
