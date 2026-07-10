// manifest-gen regenerates .tools/manifest.json (the committed source of
// truth) from the staging index produced by mirror-go + build-tools. Adds
// asset_url from the Release tag and omits skipped cells (absent from the
// index). Usage: manifest-gen <release-tag>   (or set RELEASE_TAG).
package main

import (
	"fmt"
	"os"

	"github.com/onokonem/mise-repo/internal/provider"
)

func main() {
	root, err := provider.RepoRoot()
	if err != nil {
		fail(err)
	}
	tag := releaseTag()

	cells, err := provider.ReadIndex(root)
	if err != nil {
		fail(err)
	}
	if len(cells) == 0 {
		fail(fmt.Errorf("staging index %s is empty; run mirror-go + build-tools first", provider.IndexPath(root)))
	}

	n, err := provider.WriteManifest(root, tag)
	if err != nil {
		fail(err)
	}
	fmt.Printf("[manifest-gen] wrote %s: %d cells, release_tag=%s\n", provider.ManifestPath(root), n, tag)
}

func releaseTag() string {
	if len(os.Args) > 1 && os.Args[1] != "" {
		return os.Args[1]
	}
	if t := os.Getenv("RELEASE_TAG"); t != "" {
		return t
	}
	fail(fmt.Errorf("usage: manifest-gen <release-tag> (or set RELEASE_TAG)"))
	return ""
}

func fail(err error) { fmt.Fprintln(os.Stderr, "[manifest-gen] error:", err); os.Exit(1) }
