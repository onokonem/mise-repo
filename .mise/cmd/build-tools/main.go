// build-tools builds every tool listed in versions.toml via
// `go install <path>@<version>` — the same method the prior Dockerfile provider
// used — once per (tool, buildgo, os, arch) cell, with CGO_ENABLED=0 for static
// cross-compilation. A failing cell is logged + skipped (non-fatal) and simply
// omitted from the index so manifest-gen never advertises it.
//
// Add a tool by adding a [tools.<name>] subtable in versions.toml — no code
// change here.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/onokonem/mise-repo/internal/provider"
)

func main() {
	root, err := provider.RepoRoot()
	if err != nil {
		fail(err)
	}
	v, err := provider.LoadVersions(filepath.Join(root, "versions.toml"))
	if err != nil {
		fail(err)
	}

	// Empty cwd so `go install path@version` runs in module-aware mode (no
	// go.mod above it). Reused across cells.
	scratch, err := os.MkdirTemp("", "build-tools-*")
	if err != nil {
		fail(err)
	}
	defer os.RemoveAll(scratch)

	for _, buildgo := range v.GoVersions {
		fullver, err := v.GoFullVer(buildgo)
		if err != nil {
			fail(err)
		}
		logf("installing bootstrap go %s (go%s) via upstream mise go plugin", buildgo, fullver)
		if err := provider.Run(root, "mise", "install", "go@"+fullver); err != nil {
			fail(fmt.Errorf("mise install go@%s: %w", fullver, err))
		}

		for _, name := range v.SortedBuilt() {
			for _, spec := range v.Built[name] {
				for _, goos := range provider.Oses {
					for _, arch := range provider.Arches {
						if err := buildCell(scratch, root, name, spec, buildgo, fullver, goos, arch); err != nil {
							fail(err)
						}
					}
				}
			}
		}
	}
	logf("build-tools complete")
}

func buildCell(scratch, root, name string, spec provider.BuiltTool, buildgo, fullver, goos, arch string) error {
	stem := strings.TrimPrefix(spec.Version, "v")
	dir := filepath.Join(provider.ArtifactsDir(root), name, goos+"-"+arch, "go"+buildgo)
	asset := fmt.Sprintf("%s-%s-go%s.%s-%s.tar.gz", spec.Binary, stem, buildgo, goos, arch)
	out := filepath.Join(dir, asset)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	logf("  build %s %s-%s (go%s)", name, goos, arch, buildgo)
	// Non-fatal cell: capture output, skip on failure (task 3.4).
	// GOBIN=dir makes `go install` drop the binary straight in the cell dir.
	cmd := exec.Command("mise", "exec", "go@"+fullver, "--", "env",
		"GOOS="+goos, "GOARCH="+arch, "CGO_ENABLED=0", "GOBIN="+dir,
		"go", "install", "-trimpath", "-ldflags=-s -w", spec.Path+"@"+spec.Version)
	cmd.Dir = scratch
	buildLog, err := os.Create(filepath.Join(dir, "build.log"))
	if err != nil {
		return err
	}
	defer buildLog.Close()
	cmd.Stdout = buildLog
	cmd.Stderr = buildLog
	if err := cmd.Run(); err != nil {
		logf("  cell skipped (build failed, see %s): %s %s-%s go%s", filepath.Join(dir, "build.log"), name, goos, arch, buildgo)
		return nil
	}

	bin := filepath.Join(dir, spec.Binary)
	if _, err := os.Stat(bin); err != nil {
		logf("  cell skipped (binary not produced, see %s): %s %s-%s go%s", filepath.Join(dir, "build.log"), name, goos, arch, buildgo)
		return nil
	}
	// Package the binary into a tarball for uniform extraction.
	if err := provider.Run(dir, "tar", "-czf", out, spec.Binary); err != nil {
		return fmt.Errorf("tar %s: %w", asset, err)
	}
	if err := os.Remove(bin); err != nil {
		return err
	}
	got, err := provider.SHA256File(out)
	if err != nil {
		return err
	}
	return provider.AppendIndex(root, provider.StageCell{
		Tool:        name,
		Version:     spec.CleanKey(),
		ResolvedTag: spec.Version,
		Buildgo:     buildgo,
		OS:          goos,
		Arch:        arch,
		Asset:       asset,
		SHA256:      got,
		Path:        out,
	})
}

func fail(err error) { fmt.Fprintln(os.Stderr, "[build-tools] error:", err); os.Exit(1) }
func logf(format string, args ...any) {
	fmt.Printf("[build-tools] "+format+"\n", args...)
}
