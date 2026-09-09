package marketplace

import (
	"encoding/json"
	"fmt"
	"net/http"
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

type Client struct {
	httpClient *http.Client

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	index   *Index
	fetched time.Time
}

func NewClient() *Client {
	return &Client{
		httpClient: httpx.NewClient(30 * time.Second),
		cache:      make(map[string]cacheEntry),
	}
}

// GetIndex fetches and caches the index from indexURL (empty = DefaultIndexURL).
func (c *Client) GetIndex(indexURL string) (*Index, error) {
	url := strings.TrimSpace(indexURL)
	if url == "" {
		url = DefaultIndexURL
	}

	c.mu.Lock()
	if entry, ok := c.cache[url]; ok && time.Since(entry.fetched) < cacheTTL {
		c.mu.Unlock()
		return entry.index, nil
	}
	c.mu.Unlock()

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace index returned status: %d", resp.StatusCode)
	}

	var index Index
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("invalid marketplace index: %w", err)
	}

	c.mu.Lock()
	c.cache[url] = cacheEntry{index: &index, fetched: time.Now()}
	c.mu.Unlock()

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
