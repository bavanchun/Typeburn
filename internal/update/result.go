package update

import "time"

// Release is the subset of the GitHub Releases API response used for update checks.
type Release struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Name        string    `json:"name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
}

// Result holds the outcome of an update check. It is cached to disk between runs.
type Result struct {
	SchemaVersion    int       `json:"schema_version"`
	Current          string    `json:"current"`
	Latest           string    `json:"latest"`
	UpgradeAvailable bool      `json:"upgrade_available"`
	ReleaseURL       string    `json:"release_url"`
	CheckedAt        time.Time `json:"checked_at"`
}
