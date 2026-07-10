# mise-repo

Self-hosted binary provider + builder for the **tools** mise plugin — a vfox
backend (multi-tool) plugin serving `go` plus 27 built Go tools (golangci-lint,
buf, govulncheck, protoc-gen-go, mockery, ...) across the matrix
`{1.25, 1.26, 1.27-beta} x {linux, darwin} x {amd64, arm64}`. Built tools come
from the prior Dockerfile provider (`Tools.txt`), built via `go install`.

## Layout

```
.tools/                      the plugin (installed from this subdir)
  metadata.lua               plugin identity (multi-tool backend plugin)
  config.lua                 provider identity, base URL, matrix axes, naming
  manifest.json              source of truth — cells + sha256 + asset urls
  hooks/                     backend hooks: BackendListVersions / BackendInstall
                             / BackendExecEnv + util + manifest loader
manifest.schema.json         JSON Schema for manifest.json
versions.toml                the ONE review knob: [tools.go] mirror map + [tools.<built>]
.mise/cmd/ + .mise/internal/  the Go pipeline (mirror-go, build-tools, ...), run via `go run`
.mise/go.mod / go.sum        Go module (dep: BurntSushi/toml for versions.toml)
.mise.toml                   provider tasks (clean, mirror-go, build-tools, publish)
.github/workflows/sandbox.yml  weekly/manual full publish on one linux runner
consumer/                    template to drop into a consuming repo
```

## Adjusting the matrix

Edit [versions.toml](versions.toml) — that single file pins every tool:
mirrored Go tags under `[tools.go]`, and one `[tools.<name>]` subtable per
built tool (`version`, `source`, `package`, `binary`). Add a tool by adding a
subtable; bump a value and the next publish moves that cell. No other code
changes. Each built tool fans out across the full matrix, so 20 tools = 240
built cells (+ 12 go) per release.

## Build & publish (provider)

The pipeline is Go, run via `go run`. Install Go once (latest stable, from
mise's standard source):

```sh
mise use -g go@latest
mise run sandbox   # clean -> mirror-go -> build-tools -> publish
```

`publish` creates a `sandbox-<YYYYMMDD>-<short-sha>` GitHub Release, uploads all
non-skipped assets, regenerates `.tools/manifest.json`, and commits it. Run
`mise run prune` to drop stale sandbox releases.

## Consume

See [consumer/README.md](consumer/README.md). Short version:

```sh
mise plugin install tools \
  "git::https://github.com/onokonem/mise-repo.git//.tools"
```

## Notes / scope

- Go tarballs are **mirrored** from upstream (byte-identical, sha256-verified);
  not built. golangci-lint v2 is **built** per build-go cell, `CGO_ENABLED=0`.
- A failing linter cell (e.g. under a beta toolchain) is skipped and omitted
  from the manifest — the matrix stays best-effort, never all-or-nothing.
- No artifact signing/attestation yet (future work). Windows is out of scope.
