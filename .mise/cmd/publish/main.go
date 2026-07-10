// publish attaches all non-skipped artifacts to a new GitHub Release, then
// regenerates + commits .tools/manifest.json from that release. On any partial
// upload the release is deleted and the manifest is left untouched (no drift).
package main

import (
	"fmt"
	"os"
	"time"

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

	// HEAD must be on the remote so the release's --target commitish resolves;
	// GitHub returns 422 ("target_commitish is invalid") for unpushed SHAs.
	if err := provider.Run(root, "git", "push"); err != nil {
		fail(fmt.Errorf("git push before release: %w", err))
	}
	sha, _ := provider.RunCapture(root, "git", "rev-parse", "HEAD")
	if err := provider.Run(root, "gh", "release", "create", tag,
		"--title", tag,
		"--notes", fmt.Sprintf("tools plugin assets (%d cells). Regenerated manifest attached.", len(cells)),
		"--target", sha); err != nil {
		// Resumable publish: if the release already exists (e.g. a prior run
		// created it before failing), continue and let `release upload --clobber`
		// reconcile assets instead of aborting the whole run.
		if verr := provider.Run(root, "gh", "release", "view", tag); verr != nil {
			fail(fmt.Errorf("gh release create: %w", err))
		}
		logf("release %s already exists; reconciling assets", tag)
	}

	uploadArgs := append([]string{"release", "upload", tag, "--clobber"}, paths...)
	// GitHub intermittently 502s when pushing hundreds of assets; --clobber makes
	// re-uploading safe, so retry the whole upload a few times before giving up.
	const uploadAttempts = 4
	var uploadErr error
	for attempt := 1; attempt <= uploadAttempts; attempt++ {
		if err := provider.Run(root, "gh", uploadArgs...); err != nil {
			uploadErr = err
			logf("upload attempt %d/%d failed: %v", attempt, uploadAttempts, err)
			if attempt < uploadAttempts {
				time.Sleep(time.Duration(attempt) * 10 * time.Second)
			}
			continue
		}
		uploadErr = nil
		break
	}
	if uploadErr != nil {
		// Transient upload errors are recoverable: the release is kept and
		// `mise run publish` is idempotent — re-running re-uploads with
		// --clobber. The manifest only regenerates after a full successful
		// upload, so a failed upload cannot cause drift.
		fail(fmt.Errorf("release upload failed after %d attempts (re-run to resume): %w", uploadAttempts, uploadErr))
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
	if err := provider.Run(root, "git", "push"); err != nil {
		logf("git push reported error (manifest commit may not be on remote): %v", err)
	}
	logf("published %s; prune old sandbox-* releases with the prune task", tag)
}

func fail(err error) { fmt.Fprintln(os.Stderr, "[publish] error:", err); os.Exit(1) }
func logf(format string, args ...any) {
	fmt.Printf("[publish] "+format+"\n", args...)
}
