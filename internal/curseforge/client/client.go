package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	urlpkg "net/url"
	"sync"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/curseforge/client/types"
)

const (
	getGameVersionsPath    = "/v2/games/%s/versions"
	minecraftModSearchPath = "/v1/mods/search"
	getCategoriesPath      = "/v1/categories"
	cacheTTL               = 5 * time.Minute
)

type CurseClient interface {
	GetGameVersions(ctx context.Context, gameID string) (versions *types.GameVersionsResponse, err error)
	GetCategories(ctx context.Context, gameID string) (*types.GetCategoriesResponse, error)
	GetMods(ctx context.Context, opts ...ModsQueryOption) (*types.ModsResponse, error)
	GetModsByIDs(ctx context.Context, req *GetModsByIdsListRequest, opts ...ModsQueryOption) (*types.ModsResponse, error)
	GetModDescription(ctx context.Context, modID int64) (*types.StringResponse, error)
	GetModFiles(ctx context.Context, modID int64) (*types.GetModFilesResponse, error)
}

type cachedResponse struct {
	response  *types.ModsResponse
	timestamp time.Time
}

type curseClient struct {
	CurseClient
	opt     Config
	c       *http.Client
	cache   map[string]cachedResponse
	cacheMu sync.RWMutex
}

func NewCurseClient(apiKey string, opts ...CfgFunc) CurseClient {
	opt := NewConfig(apiKey, opts...)
	return &curseClient{
		opt:   *opt,
		c:     opt.NewHTTPClient(),
		cache: make(map[string]cachedResponse),
	}
}

func (c *curseClient) GetGameVersions(ctx context.Context, gameID string) (versions *types.GameVersionsResponse, err error) {
	log := c.opt.log
	url := c.buildRequestPath(fmt.Sprintf(getGameVersionsPath, urlpkg.PathEscape(gameID)), ApiQueryParams{})
	if c.opt.debug {
		log.Printf("GetGameVersions.Begin id: %s (url: %s)", gameID, url)
	}
	req, err := c.opt.NewGetRequest(url)
	if err != nil {
		return nil, fmt.Errorf("creating get game versions request: %w", err)
	}

	var gv types.GameVersionsResponse
	if err := c.executeRequest(req.WithContext(ctx), "get game versions", &gv); err != nil {
		return nil, fmt.Errorf("executing get game versions request: %w", err)
	}

	return &gv, nil
}

func (c *curseClient) GetCategories(ctx context.Context, gameID string) (*types.GetCategoriesResponse, error) {
	q := ApiQueryParams{
		"gameId": gameID,
	}
	req, err := c.opt.NewGetRequest(c.buildRequestPath(getCategoriesPath, q))
	if err != nil {
		return nil, fmt.Errorf("creating get categories request: %w", err)
	}

	var resp types.GetCategoriesResponse
	if err := c.executeRequest(req.WithContext(ctx), "get categories", &resp); err != nil {
		return nil, fmt.Errorf("executing get categories request: %w", err)
	}

	return &resp, nil
}

func (c *curseClient) GetModFiles(ctx context.Context, modID int64) (*types.GetModFilesResponse, error) {
	path := fmt.Sprintf("/v1/mods/%d/files", modID)
	req, err := c.opt.NewGetRequest(path)
	if err != nil {
		return nil, fmt.Errorf("creating get mod files request: %w", err)
	}

	var resp types.GetModFilesResponse
	if err := c.executeRequest(req.WithContext(ctx), "get mod files", &resp); err != nil {
		return nil, fmt.Errorf("executing get mod files request: %w", err)
	}

	return &resp, nil
}

func (c *curseClient) GetMods(ctx context.Context, opts ...ModsQueryOption) (*types.ModsResponse, error) {
	q := ApiQueryParams{}
	for _, o := range opts {
		o(q)
	}

	cacheKey := q.QueryString()
	c.cacheMu.RLock()
	if entry, ok := c.cache[cacheKey]; ok && time.Since(entry.timestamp) < cacheTTL && entry.response != nil {
		c.cacheMu.RUnlock()
		return entry.response, nil
	}
	c.cacheMu.RUnlock()

	var result types.ModsResponse
	req, err := c.opt.NewGetRequest(c.buildRequestPath(minecraftModSearchPath, q))
	if err != nil {
		err = fmt.Errorf("creating get minecraft mods request: %w", err)
		return &result, types.Wrap(err, "failed to create request object", -1)
	}

	var mv types.ModsResponse
	if err := c.executeRequest(req.WithContext(ctx), "get mods", &mv); err != nil {
		err = fmt.Errorf("executing get minecraft mods request: %w", err)
		return &result, types.Wrap(err, "failed to execute request", -1)
	}

	c.cacheMu.Lock()
	c.cache[cacheKey] = cachedResponse{
		response:  &mv,
		timestamp: time.Now(),
	}
	if len(c.cache) > 200 {
		for k := range c.cache {
			delete(c.cache, k)
			break
		}
	}
	c.cacheMu.Unlock()

	return &mv, nil
}

func (c *curseClient) GetModsByIDs(ctx context.Context, filter *GetModsByIdsListRequest, opts ...ModsQueryOption) (*types.ModsResponse, error) {
	if filter == nil || len(filter.ModIds) < 1 {
		return nil, fmt.Errorf("%w: invalid filter, mod ids required (%#v)", types.ErrInvalidRequestParams, filter)
	}
	q := ApiQueryParams{}
	for _, o := range opts {
		o(q)
	}
	var result types.ModsResponse
	req, err := c.opt.NewPostRequest("/v1/mods", filter)
	if err != nil {
		err = fmt.Errorf("creating get minecraft mods request: %w", err)
		return &result, types.Wrap(err, "failed to create request object", -1)
	}

	var mv types.ModsResponse
	if err := c.executeRequest(req.WithContext(ctx), "get mods by ids", &mv); err != nil {
		return &result, types.Wrap(err, "failed to execute request", -1)
	}

	return &mv, nil
}

func (c *curseClient) GetModDescription(ctx context.Context, modID int64) (*types.StringResponse, error) {
	path := fmt.Sprintf("/v1/mods/%d/description", modID)
	req, err := c.opt.NewGetRequest(path)
	if err != nil {
		return nil, fmt.Errorf("creating get mod description request: %w", err)
	}

	var resp types.StringResponse
	if err := c.executeRequest(req.WithContext(ctx), "get mod description", &resp); err != nil {
		return nil, fmt.Errorf("executing get mod description request: %w", err)
	}

	return &resp, nil
}

func (c *curseClient) buildRequestPath(path string, q ApiQueryParams) string {
	return fmt.Sprintf("%s?%s", path, q.QueryString())
}

// maxAPIResponseBytes caps a single CurseForge API response body.
const maxAPIResponseBytes = 16 << 20

// readAPIResponseBody reads the whole response body exactly once and
// always closes it. The body is capped so a hostile or broken endpoint
// cannot exhaust memory.
func readAPIResponseBody(res *http.Response) ([]byte, error) {
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, maxAPIResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxAPIResponseBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxAPIResponseBytes)
	}
	return b, nil
}

// parseResponse parses an already-read response body. Reading and
// closing the body is the caller's job (see readAPIResponseBody).
func (c *curseClient) parseResponse(statusCode int, body []byte, step string, out types.CurseforgeAPIResponse) error {
	if statusCode/100 != 2 {
		return types.Wrap(
			fmt.Errorf("%s request failed with status code %d", step, statusCode),
			string(body),
			statusCode,
		)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return types.Wrap(
			fmt.Errorf("decoding get game versions response: %w", err),
			string(body),
			statusCode,
		)
	}
	out.SetRawResponseBody(string(body))

	return nil
}

func (c *curseClient) executeRequest(req *http.Request, step string, out types.CurseforgeAPIResponse) error {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	debug := c.opt.debug

	// Request/response bodies are only captured for logging when the
	// debug flag is on; the default log line carries just the method,
	// URL path, status and byte count.
	var reqBody []byte
	if debug && req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("reading api request body: %w", err)
		}
		reqBody = b
		req.Body = io.NopCloser(bytes.NewReader(b))
	}

	res, err := c.c.Do(req)
	if err != nil {
		err = fmt.Errorf("executing get minecraft mods request: %w", err)
		if debug {
			slog.With("step", step, "method", req.Method, "url", req.URL.String(), "request_body", string(reqBody), "error", err).Error("APIRequest")
		} else {
			slog.With("step", step, "method", req.Method, "path", req.URL.Path, "error", err).Error("APIRequest")
		}
		return types.Wrap(err, "failed to execute request", -1)
	}

	resBody, err := readAPIResponseBody(res)
	if err != nil {
		return fmt.Errorf("reading api response body: %w", err)
	}

	if debug {
		slog.With("step", step, "method", req.Method, "url", req.URL.String(), "request_body", string(reqBody), "status_code", res.StatusCode, "response_body", string(resBody)).Info("APIRequest")
	} else {
		slog.With("step", step, "method", req.Method, "path", req.URL.Path, "status_code", res.StatusCode, "response_bytes", len(resBody)).Info("APIRequest")
	}

	if err := c.parseResponse(res.StatusCode, resBody, step, out); err != nil {
		return fmt.Errorf("parsing api response response: %w", err)
	}

	return nil
}
