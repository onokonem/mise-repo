package provider

// Static provider identity + matrix axes. These are structural (not version
// data, which lives in versions.toml). Keep in sync with .tools/config.lua.

const (
	Owner          = "onokonem"
	Repo           = "mise-repo"
	DefaultBuildgo = "1.26" // build-go for golangci-lint when a request omits -go<buildgo>
)

// Oses / Arches are the supported matrix axes (linux+darwin only; no windows).
var (
	Oses   = []string{"linux", "darwin"}
	Arches = []string{"amd64", "arm64"}
)

// ReleaseBaseURL is the GitHub Releases download root (no tag appended).
func ReleaseBaseURL() string {
	return "https://github.com/" + Owner + "/" + Repo + "/releases/download"
}

// StageCell is one entry in the staging index (artifacts/.index.jsonl): the
// result of a mirror/build step before it has a Release asset_url. Path is the
// local artifact file for upload.
type StageCell struct {
	Tool        string `json:"tool"`
	Version     string `json:"version"`      // clean key, e.g. "1.26"
	ResolvedTag string `json:"resolved_tag"` // concrete upstream/built tag
	Buildgo     string `json:"buildgo"`      // go toolchain that produced the asset
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Asset       string `json:"asset"`  // Release asset filename
	SHA256      string `json:"sha256"` // sha256 of the asset file
	Path        string `json:"path"`   // local file path (staging only)
}

// Cell is one published entry in manifest.json (matches manifest.schema.json).
type Cell struct {
	Tool        string `json:"tool"`
	Version     string `json:"version"`
	ResolvedTag string `json:"resolved_tag"`
	Buildgo     string `json:"buildgo"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Asset       string `json:"asset"`
	AssetURL    string `json:"asset_url"`
	SHA256      string `json:"sha256"`
}

// Manifest is the committed source of truth shipped inside the plugin clone.
type Manifest struct {
	ReleaseTag     string `json:"release_tag"`
	BaseURL        string `json:"base_url"`
	GeneratedAt    string `json:"generated_at"`
	DefaultBuildgo string `json:"default_buildgo"`
	Cells          []Cell `json:"cells"`
}
