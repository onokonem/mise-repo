// publish attaches all non-skipped artifacts to a new GitHub Release, then
// regenerates + commits .tools/manifest.json from that release. On any partial
// upload the release is deleted and the manifest is left untouched (no drift).
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
	if err := provider.Run(root, "gh", "auth", "status"); err != nil {
		fail(fmt.Errorf("gh is not authenticated: %w", err))
	}

	cells, err := provider.ReadIndex(root)
	if err != nil {
		fail(err)
	}
	if len(cells) == 0 {
		fail(fmt.Errorf("staging index empty; run mirror-go + build-tools first"))
	}

	tag, err := provider.ReleaseTag(root)
	if err != nil {
		fail(err)
	}
	paths := make([]string, 0, len(cells))
	for _, c := range cells {
		if _, err := os.Stat(c.Path); err != nil {
			fail(fmt.Errorf("indexed asset missing on disk: %s (%w)", c.Path, err))
		}
		paths = append(paths, c.Path)
	}

	logf("publishing %d cells as release %s", len(cells), tag)

	sha, _ := provider.RunCapture(root, "git", "rev-parse", "HEAD")
	if err := provider.Run(root, "gh", "release", "create", tag,
		"--title", tag,
		"--notes", fmt.Sprintf("tools plugin assets (%d cells). Regenerated manifest attached.", len(cells)),
		"--target", sha); err != nil {
		fail(fmt.Errorf("gh release create: %w", err))
	}

	uploadArgs := append([]string{"release", "upload", tag, "--clobber"}, paths...)
	if err := provider.Run(root, "gh", uploadArgs...); err != nil {
		logf("upload failed; aborting release %s to avoid manifest drift", tag)
		_ = provider.Run(root, "gh", "release", "delete", tag, "--yes", "--cleanup-tag")
		fail(fmt.Errorf("release aborted; manifest not regenerated: %w", err))
	}

	n, err := provider.WriteManifest(root, tag)
	if err != nil {
		fail(err)
	}
	if err := provider.Run(root, "git", "add", ".tools/manifest.json"); err != nil {
		fail(err)
	}
	if err := provider.Run(root, "git", "commit", "-m",
		fmt.Sprintf("chore(sandbox): publish manifest for %s (%d cells)", tag, n)); err != nil {
		logf("git commit reported error (manifest may be unchanged): %v", err)
	}
	logf("published %s; prune old sandbox-* releases with the prune task", tag)
}

func fail(err error) { fmt.Fprintln(os.Stderr, "[publish] error:", err); os.Exit(1) }
func logf(format string, args ...any) {
	fmt.Printf("[publish] "+format+"\n", args...)
}
