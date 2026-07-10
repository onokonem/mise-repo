// prune deletes prior sandbox-* releases/tags (R1 bloat mitigation). Keeps the
// release named on the command line, or the one recorded in the committed
// manifest. Usage: prune [tag-to-keep].
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/onokonem/mise-repo/internal/provider"
)

func main() {
	root, err := provider.RepoRoot()
	if err != nil {
		fail(err)
	}
	keep := keepTag(root, os.Args[1:])

	logf("pruning sandbox-* releases (keeping: %s)", keep)

	tags, err := provider.GHReleaseList()
	if err != nil {
		fail(err)
	}
	for _, t := range tags {
		if !strings.HasPrefix(t, "sandbox-") {
			continue
		}
		if t == keep {
			logf("  keep %s", t)
			continue
		}
		logf("  delete %s", t)
		if err := provider.Run(root, "gh", "release", "delete", t, "--yes", "--cleanup-tag"); err != nil {
			fail(fmt.Errorf("delete %s: %w", t, err))
		}
	}
	logf("prune complete")
}

func keepTag(root string, args []string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	if b, err := os.ReadFile(provider.ManifestPath(root)); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, `"release_tag"`) {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.Trim(strings.TrimSpace(parts[1]), `",`)
				}
			}
		}
	}
	return ""
}

func fail(err error) { fmt.Fprintln(os.Stderr, "[prune] error:", err); os.Exit(1) }
func logf(format string, args ...any) {
	fmt.Printf("[prune] "+format+"\n", args...)
}
