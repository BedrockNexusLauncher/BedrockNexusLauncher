// Package mcpedl provides an experimental client for the MCPEDL community
// catalogue API (api.mcpedl.com) so Bedrock addons can be searched and their
// files downloaded through the launcher's existing download/import pipeline.
package mcpedl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/httpx"
)

const (
	apiBase        = "https://api.mcpedl.com/api/submissions"
	requestTimeout = 30 * time.Second
	defaultPerPage = 30
)

// DownloadFile is a single downloadable file of a submission. DownloadURL
// points at the direct file (usually .mcpack/.mcaddon/.mcworld) and can be fed
// into the launcher's StartFileDownload pipeline.
type DownloadFile struct {
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	DownloadURL string `json:"downloadUrl"`
}

// Submission is a catalogue entry (addon, map, texture pack, ...).
type Submission struct {
	Slug          string         `json:"slug"`
	Title         string         `json:"title"`
	Summary       string         `json:"summary"`
	Image         string         `json:"image"`
	URL           string         `json:"url"`
	Source        string         `json:"source"`
	DownloadCount int            `json:"downloadCount"`
	AverageRating string         `json:"average_rating"`
	UpdateDate    string         `json:"update_date"`
	Downloads     []DownloadFile `json:"downloads"`
}

type apiResponse struct {
	Status string       `json:"status"`
	Data   []Submission `json:"data"`
}

// Search queries the catalogue. Empty query returns the newest submissions.
// page starts at 1; the upstream page size is controlled via per_page.
func Search(query string, page int) ([]Submission, error) {
	u, err := url.Parse(apiBase)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if query != "" {
		q.Set("s", query)
	}
	q.Set("per_page", strconv.Itoa(defaultPerPage))
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	u.RawQuery = q.Encode()

	client := httpx.NewClient(requestTimeout)
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status: %d", resp.StatusCode)
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Status != "" && result.Status != "success" {
		return nil, fmt.Errorf("api returned status: %s", result.Status)
	}
	return result.Data, nil
}
