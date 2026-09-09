// Package marketplace reads a community "marketplace" index: a single JSON
// document (index.json) hosted on GitHub (repo + Releases) describing
// downloadable Bedrock addons, skin packs, worlds, etc. No server is involved.
package marketplace

// Item is one downloadable entry in the marketplace index.
type Item struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Author      string   `json:"author"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Screenshots []string `json:"screenshots"`
	Tags        []string `json:"tags"`
	Type        string   `json:"type"`
	Homepage    string   `json:"homepage"`
	Versions    []File   `json:"versions"`
}

// File is one downloadable version of an Item.
type File struct {
	Version     string `json:"version"`
	Date        string `json:"date"`
	FileType    string `json:"fileType"`
	FileName    string `json:"fileName"`
	DownloadURL string `json:"downloadUrl"`
	Size        int64  `json:"size"`
	Changelog   string `json:"changelog"`
}

// Index is the root document of index.json.
type Index struct {
	SchemaVersion int    `json:"schemaVersion"`
	UpdatedAt     string `json:"updatedAt"`
	Items         []Item `json:"items"`
}
