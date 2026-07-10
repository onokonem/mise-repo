// Package provider holds the shared types and helpers for the `tools` plugin
// provider pipeline (.mise/cmd/*). All tool versions are pinned in versions.toml
// at the repo root — nothing is resolved online at build time.
package provider

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// versions.toml shape. [tools] is mixed: tools.go is a key->tag map (mirrored),
// every other tools.<name> is a built-tool subtable. Decoded via toml.Primitive
// so the two shapes can coexist under one table.
type tomlMatrix struct {
	GoVersions []string `toml:"go_versions"`
}

// BuiltTool is one published version of a tool under [[tools.<name>]], built via
// `go install <path>@<version>` (same method the prior Dockerfile provider used).
type BuiltTool struct {
	Version string `toml:"version"` // tag or commit hash (recorded as resolved_tag)
	Path    string `toml:"path"`    // go install import path
	Binary  string `toml:"binary"`  // resulting binary name
	Key     string `toml:"key"`     // optional clean selector key; defaults to CleanVer(Version). Set when two versions share a major.minor (e.g. v1.6.0 vs v1.6.2).
}

// CleanKey is the manifest version field for this entry: explicit Key if set,
// else CleanVer(Version).
func (b BuiltTool) CleanKey() string {
	if b.Key != "" {
		return b.Key
	}
	return CleanVer(b.Version)
}

type rawDoc struct {
	Matrix tomlMatrix                `toml:"matrix"`
	Tools  map[string]toml.Primitive `toml:"tools"`
}

// Versions is the parsed, pinned matrix from versions.toml.
type Versions struct {
	GoVersions []string               // clean selector keys, e.g. "1.26"
	GoTags     map[string]string      // mirrored go key -> upstream tag
	Built      map[string][]BuiltTool // built tool name -> one entry per published version
}

// LoadVersions reads versions.toml from path.
func LoadVersions(path string) (*Versions, error) {
	var raw rawDoc
	md, err := toml.DecodeFile(path, &raw)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	v := &Versions{
		GoVersions: raw.Matrix.GoVersions,
		GoTags:     map[string]string{},
		Built:      map[string][]BuiltTool{},
	}
	for name, prim := range raw.Tools {
		if name == "go" {
			var m map[string]string
			if err := md.PrimitiveDecode(prim, &m); err != nil {
				return nil, fmt.Errorf("decode tools.go: %w", err)
			}
			v.GoTags = m
			continue
		}
		// [[tools.<name>]] array of tables -> one entry per published version.
		var entries []BuiltTool
		if err := md.PrimitiveDecode(prim, &entries); err != nil {
			return nil, fmt.Errorf("decode tools.%s: %w", name, err)
		}
		seen := map[string]bool{}
		for _, bt := range entries {
			if bt.Version == "" || bt.Path == "" || bt.Binary == "" {
				return nil, fmt.Errorf("tools.%s entry must set version, path, binary", name)
			}
			key := bt.CleanKey()
			if seen[key] {
				return nil, fmt.Errorf("tools.%s: duplicate version key %q (from %s); set a unique `key` or use distinct major.minor", name, key, bt.Version)
			}
			seen[key] = true
		}
		v.Built[name] = entries
	}
	if len(v.GoTags) == 0 {
		return nil, fmt.Errorf("versions.toml has no [tools.go] mirror map")
	}
	return v, nil
}

// GoTag returns the pinned upstream Go tag for a clean key.
func (v *Versions) GoTag(key string) (string, error) {
	t, ok := v.GoTags[key]
	if !ok {
		return "", fmt.Errorf("no pinned Go tag for %q (add it under [tools.go] in versions.toml)", key)
	}
	return t, nil
}

// GoFullVer returns the upstream version ("1.26.5") for `mise install go@<ver>`.
func (v *Versions) GoFullVer(key string) (string, error) {
	t, err := v.GoTag(key)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(t, "go"), nil
}

// SortedBuilt returns built-tool names in deterministic order.
func (v *Versions) SortedBuilt() []string {
	names := make([]string, 0, len(v.Built))
	for n := range v.Built {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// CleanVer derives the clean selector key (major.minor) from a version tag.
// "v2.1.0" -> "2.1"; "2025.1.1" -> "2025.1".
func CleanVer(tag string) string {
	s := strings.TrimPrefix(tag, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return s
}
