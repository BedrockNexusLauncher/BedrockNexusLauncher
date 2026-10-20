package marketplace

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/httpx"
)

// DefaultIndexURL points at the community marketplace index. Any GitHub repo
// following the schema in marketplace/README.md can be substituted by passing
// another URL to the RPCs.
const DefaultIndexURL = "https://cdn.jsdelivr.net/gh/BedrockNexusLauncher/Marketplace@main/index.json"

const cacheTTL = 5 * time.Minute

// maxIndexBytes caps the index document we are willing to decode
// (matching the 8 MiB body cap the gdkversions client uses).
const maxIndexBytes = 8 << 20

type Client struct {
	httpClient *http.Client

	mu       sync.Mutex
	cache    map[string]cacheEntry
	inflight map[string]*indexFetch
}

type cacheEntry struct {
	index   *Index
	fetched time.Time
}

// indexFetch is one in-flight index fetch. Concurrent GetIndex calls
// for the same URL wait on done and share the single outcome instead
// of each issuing their own request for the same document.
type indexFetch struct {
	done  chan struct{}
	index *Index
	err   error
}

func NewClient() *Client {
	return &Client{
		httpClient: httpx.NewClient(30 * time.Second),
		cache:      make(map[string]cacheEntry),
		inflight:   make(map[string]*indexFetch),
	}
}

// GetIndex fetches and caches the index from indexURL (empty = DefaultIndexURL).
//
// The index URL is caller-supplied (user-editable in the frontend) and
// every item in the index becomes a download candidate, so the URL is
// validated before any fetch: https only, no credentials, and no
// loopback/private/link-local IP hosts (see httpx.ValidatePublicHTTPSURL).
func (c *Client) GetIndex(indexURL string) (*Index, error) {
	url := strings.TrimSpace(indexURL)
	if url == "" {
		url = DefaultIndexURL
	}

	parsed, err := neturl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("invalid marketplace index URL: %w", err)
	}
	if err := httpx.ValidatePublicHTTPSURL(parsed); err != nil {
		return nil, fmt.Errorf("invalid marketplace index URL: %w", err)
	}

	c.mu.Lock()
	if entry, ok := c.cache[url]; ok && time.Since(entry.fetched) < cacheTTL {
		c.mu.Unlock()
		return entry.index, nil
	}
	if pending, ok := c.inflight[url]; ok {
		// Another goroutine is already fetching this URL; share its
		// result (single-flight) instead of fetching a duplicate.
		c.mu.Unlock()
		<-pending.done
		return pending.index, pending.err
	}
	pending := &indexFetch{done: make(chan struct{})}
	c.inflight[url] = pending
	c.mu.Unlock()

	pending.index, pending.err = c.fetchIndex(url)

	c.mu.Lock()
	if pending.err == nil {
		c.cache[url] = cacheEntry{index: pending.index, fetched: time.Now()}
	}
	delete(c.inflight, url)
	c.mu.Unlock()
	close(pending.done)

	return pending.index, pending.err
}

// fetchIndex performs the actual index request for an already
// validated URL.
func (c *Client) fetchIndex(url string) (*Index, error) {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace index returned status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxIndexBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading marketplace index: %w", err)
	}
	if int64(len(data)) > maxIndexBytes {
		return nil, fmt.Errorf("marketplace index exceeds %d bytes", maxIndexBytes)
	}

	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("invalid marketplace index: %w", err)
	}

	return &index, nil
}

// GetItem returns a single item by id from the (cached) index.
func (c *Client) GetItem(indexURL string, id string) (*Item, error) {
	index, err := c.GetIndex(indexURL)
	if err != nil {
		return nil, err
	}
	for i := range index.Items {
		if index.Items[i].ID == id {
			return &index.Items[i], nil
		}
	}
	return nil, fmt.Errorf("marketplace item not found: %s", id)
}
