// mirror-go mirrors the 12 official Go tarballs (3 versions x 4 os/arch) into
// artifacts/, verifying each against the upstream go.dev sha256. Records every
// cell into the staging index. Tags are pinned in versions.toml; the go.dev
// index is fetched only for checksum verification.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/onokonem/mise-repo/internal/provider"
)

// go.dev/dl/?mode=json&include=all shape (only what we need).
type goFile struct {
	Filename string `json:"filename"`
	SHA256   string `json:"sha256"`
}
type goRelease struct {
	Version string   `json:"version"`
	Files   []goFile `json:"files"`
}

func main() {
	root, err := provider.RepoRoot()
	if err != nil {
		fail(err)
	}
	v, err := provider.LoadVersions(filepath.Join(root, "versions.toml"))
	if err != nil {
		fail(err)
	}

	logf("fetching upstream go.dev/dl index for checksum verification")
	releases, err := fetchGoIndex()
	if err != nil {
		fail(err)
	}

	for _, key := range v.GoVersions {
		tag, err := v.GoTag(key)
		if err != nil {
			fail(err)
		}
		logf("go %s -> pinned upstream %s", key, tag)
		for _, goos := range provider.Oses {
			for _, arch := range provider.Arches {
				if err := mirrorCell(root, releases, key, tag, goos, arch); err != nil {
					fail(err)
				}
			}
		}
	}
	logf("mirror-go complete")
}

func mirrorCell(root string, releases []goRelease, key, tag, goos, arch string) error {
	asset := fmt.Sprintf("%s.%s-%s.tar.gz", tag, goos, arch)
	dir := filepath.Join(provider.ArtifactsDir(root), "go", goos+"-"+arch, key)
	out := filepath.Join(dir, asset)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	expected, err := upstreamSHA(releases, tag, asset)
	if err != nil {
		return err
	}

	url := "https://dl.google.com/go/" + asset
	logf("  download %s", asset)
	if err := download(url, out); err != nil {
		return fmt.Errorf("download %s: %w", asset, err)
	}

	got, err := provider.SHA256File(out)
	if err != nil {
		return err
	}
	if got != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", asset, expected, got)
	}

	return provider.AppendIndex(root, provider.StageCell{
		Tool:        "go",
		Version:     key,
		ResolvedTag: tag,
		Buildgo:     key,
		OS:          goos,
		Arch:        arch,
		Asset:       asset,
		SHA256:      got,
		Path:        out,
	})
}

func fetchGoIndex() ([]goRelease, error) {
	resp, err := http.Get("https://go.dev/dl/?mode=json&include=all")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r []goRelease
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return r, nil
}

func upstreamSHA(releases []goRelease, tag, asset string) (string, error) {
	for _, rel := range releases {
		if rel.Version != tag {
			continue
		}
		for _, f := range rel.Files {
			if f.Filename == asset {
				return f.SHA256, nil
			}
		}
	}
	return "", fmt.Errorf("no upstream checksum for %s (is %s a real published tag?)", asset, tag)
}

func download(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func fail(err error) { fmt.Fprintln(os.Stderr, "[mirror-go] error:", err); os.Exit(1) }
func logf(format string, args ...any) {
	fmt.Printf("[mirror-go] "+format+"\n", args...)
}
