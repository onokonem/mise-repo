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

## Binary storage

Built binaries are **not stored in git**. There is no Git-LFS and `artifacts/`
is gitignored (`.gitignore:11`) — the build tree is never committed. The
canonical home of every binary is a **GitHub Release `.tar.gz`**; the repo
carries only the manifest pointer.

**Format.** Each cell is a single stripped binary packaged as a `.tar.gz`.
Cross-compiled with `CGO_ENABLED=0 -trimpath -ldflags=-s -w` via `go install`
([build-tools/main.go:78-82](.mise/cmd/build-tools/main.go#L78-L82)), then
`tar -czf`'d alone ([build-tools/main.go:106](.mise/cmd/build-tools/main.go#L106)).
Asset name:
`<binary>-<stem>-go<buildgo>.<os>-<arch>.tar.gz`
([build-tools/main.go:67](.mise/cmd/build-tools/main.go#L67)).

**Local build output (gitignored).** Tarballs land under
`artifacts/<tool>/<os>-<arch>/go<buildgo>/<asset>.tar.gz`
([build-tools/main.go:66-68](.mise/cmd/build-tools/main.go#L66-L68)); a staging
index row per cell is appended to `artifacts/.index.jsonl`
(`{asset, sha256, path}`).

**Publish (git → GitHub).** `publish` opens a `sandbox-<YYYYMMDD>-<short-sha>`
release ([publish/main.go:45-49](.mise/cmd/publish/main.go#L45-L49)), uploads
all non-skipped tarballs with `gh release upload --clobber` — aborting and
deleting the release on any partial failure
([publish/main.go:52-53](.mise/cmd/publish/main.go#L52-L53)) — then regenerates
and commits `.tools/manifest.json`
([publish/main.go:63-67](.mise/cmd/publish/main.go#L63-L67)).

**Committed pointer (the only thing tracked).**
[.tools/manifest.json](.tools/manifest.json) records, for every cell, the
`asset`, `asset_url`, `sha256`, and `release_tag`. The base download URL lives
in [config.lua:16](.tools/config.lua#L16)
(`release_base_url = https://github.com/onokonem/mise-repo/releases/download`).

**Consumer retrieval.** On install the plugin fetches the cell's `asset_url`
via `http.download_file` ([backend_install.lua:62](.tools/hooks/backend_install.lua#L62)),
verifies `sha256` against the manifest
([backend_install.lua:65-73](.tools/hooks/backend_install.lua#L65-L73)), then
`archiver.decompress` extracts the tarball into the install path
([backend_install.lua:77](.tools/hooks/backend_install.lua#L77)).

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
