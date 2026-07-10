package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteManifest regenerates .tools/manifest.json from the staging index for the
// given release tag. Returns the number of cells written.
func WriteManifest(root, tag string) (int, error) {
	cells, err := ReadIndex(root)
	if err != nil {
		return 0, err
	}
	base := ReleaseBaseURL() + "/" + tag
	m := Manifest{
		ReleaseTag:     tag,
		BaseURL:        base,
		GeneratedAt:    time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		DefaultBuildgo: DefaultBuildgo,
		Cells:          make([]Cell, 0, len(cells)),
	}
	for _, sc := range cells {
		m.Cells = append(m.Cells, Cell{
			Tool:        sc.Tool,
			Version:     sc.Version,
			ResolvedTag: sc.ResolvedTag,
			Buildgo:     sc.Buildgo,
			OS:          sc.OS,
			Arch:        sc.Arch,
			Asset:       sc.Asset,
			AssetURL:    base + "/" + sc.Asset,
			SHA256:      sc.SHA256,
		})
	}

	out := ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return 0, err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return 0, err
	}
	b = append(b, '\n')
	if err := os.WriteFile(out, b, 0o644); err != nil {
		return 0, err
	}
	return len(m.Cells), nil
}

// GHReleaseList returns sandbox-* release tag names via the gh CLI.
func GHReleaseList() ([]string, error) {
	out, err := RunCapture("", "gh", "release", "list", "--json", "tagName", "-q", ".[].tagName")
	if err != nil {
		return nil, fmt.Errorf("gh release list: %w", err)
	}
	var tags []string
	for _, l := range splitLines(out) {
		if l != "" {
			tags = append(tags, l)
		}
	}
	return tags, nil
}
