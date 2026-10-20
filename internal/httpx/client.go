package httpx

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// checkRedirect is the redirect policy for every request made through
// this package: at most 10 hops, never a downgrade to plain http, and
// never to a non-public IP (loopback / private / link-local), which a
// hostile or compromised server could otherwise use to make the
// launcher talk to local or LAN services.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("too many redirects")
	}
	return ValidatePublicHTTPSURL(req.URL)
}

// ValidatePublicHTTPSURL requires an https URL whose host is either a
// DNS name or a public IP address.
func ValidatePublicHTTPSURL(u *url.URL) error {
	if u == nil {
		return fmt.Errorf("missing URL")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("only https URLs are allowed, got scheme %q", u.Scheme)
	}
	if u.User != nil {
		return fmt.Errorf("URLs with credentials are not allowed")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL has no host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("host %q is not a public IP address", host)
		}
	}
	return nil
}

type defaultHeaderTransport struct {
	base http.RoundTripper
}

func (t *defaultHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ApplyDefaultHeaders(req)
	return t.base.RoundTrip(req)
}

var sharedTransport http.RoundTripper = &defaultHeaderTransport{base: newBaseTransport()}

var sharedClient = &http.Client{
	Transport:     sharedTransport,
	CheckRedirect: checkRedirect,
}

func DefaultClient() *http.Client {
	return sharedClient
}

func NewClient(timeout time.Duration) *http.Client {
	client := &http.Client{
		Transport:     sharedTransport,
		CheckRedirect: checkRedirect,
	}
	if timeout > 0 {
		client.Timeout = timeout
	}
	return client
}

func Do(req *http.Request) (*http.Response, error) {
	ApplyDefaultHeaders(req)
	return sharedClient.Do(req)
}

func Get(target string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	return Do(req)
}
