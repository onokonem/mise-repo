# Design — add-mise-sandbox

## Context

Two sibling repos:
- `mise-repo` (this repo) — empty binary-provider repo. Becomes the sandbox source.
- `easyp-buf-proxy` — Go module (`go 1.26`), needs build/test/lint across the matrix.

Verified constraints:
- mise `2026.7.5` installed locally.
- mise docs: **asdf plugins are legacy** and not accepted to the registry; **vfox is the recommended custom-plugin backend** and supports **multi-tool plugins** via the `plugin:tool` format. This is the load-bearing decision (see Alternatives).
- vfox `PreInstall` returns `{url, sha256}` and mise performs the download → can point at arbitrary self-hosted URLs (GitHub Release assets).
- vfox `Available` returns structured version objects with built-in semver support.
- GitHub raw refuses files >100 MB → Go tarballs (~60–150 MB) must be Release assets.

## Goals / Non-Goals

Goals:
- One vfox plugin in `mise-repo` provides both `go` and `golangci-lint`.
- Full matrix available and selectable: 12 go cells + 12 cells per (tool, version); 27 tools / 31 versions incl. legacy ones, all pinned in versions.toml.
- Consumer selects a matrix cell per invocation with zero host-installed toolchains.
- Native-vs-cross semantics enforced.

Non-goals (deferred):
- Artifact signing/attestation (cosign/SLSA).
- Windows support (matrix is linux/darwin only).
- Replacing easyp's GitHub Actions (consumer config complements them).

## Decisions

### D1 — vfox multi-tool backend (not asdf)
One plugin registered as `tools` serves `tools:go` and `tools:golangci-lint`. Hooks receive the tool name, so shared Lua code branches per tool. No upstream name shadowing: the tool-id is `tools:go` but the executable placed on PATH is still bare `go`.

Registration (consumer, one-time):
```
mise plugin install tools "git::https://github.com/<owner>/mise-repo.git//.tools"
```
Then `[tools] "tools:go" = "1.26"` etc.

### D2 — go is mirrored; every other tool is built via `go install`
- `go`: download the 12 official Go tarballs (3 versions × 4 os/arch), repackage as Release assets. Building Go from source is pointless — Go bootstraps itself, producing bytes identical to upstream.
- built tools (27, listed in `versions.toml` under `[tools.<name>]`, sourced from the prior Dockerfile provider `Tools.txt` minus the private `yadro.*` and non-Go tools): each tool produces 12 binaries — one per `(buildgo, goos, goarch)` cell — built with the same `go install` method the Dockerfile used. Method per cell (drives the `build-tools` Go program):
  ```
  mise exec go@<buildgo> -- env GOOS=<goos> GOARCH=<goarch> CGO_ENABLED=0 GOBIN=<dir> \
    go install -trimpath -ldflags="-s -w" <path>@<version>
  ```
  `CGO_ENABLED=0` is required for static cross-compilation; all listed tools are pure Go so this is safe. A single Linux CI runner produces all cells. With N built tools the asset count is 12 (go) + 12·N.

### D3 — Suffix-encoded version strings (the selection key)
Format: `@<ver>[-go<buildgo>][-<goos>-<goarch>]`
```
go@1.26                              # host os/arch, ver 1.26
go@1.26-darwin-arm64                 # pinned cell
golangci-lint@2.12-go1.26             # host os/arch, built by go1.26
golangci-lint@2.12-go1.26-darwin-arm64   # fully pinned cell
```
Parsing in `PreInstall`: absent `-goX` (lint) → default build-go (latest stable of the three); absent `<goos>-<goarch>` → read os/arch from hook context (host). `Available` emits clean defaults (`2.12`, `2.12-go1.26`) as primary rows so `@latest` resolves; full suffix forms are opt-in.

### D4 — manifest.json is source of truth
`manifest.json` enumerates every published cell (tool, version, buildgo, os, arch, asset filename, sha256). `Available` is generated from it, and `PreInstall` looks up the asset URL + checksum from it. This guarantees the plugin never advertises a cell that has no asset and never drifts from Releases. The publish task regenerates `manifest.json` from the built artifact tree.

### D5 — native vs cross task semantics
| task   | executed binary GOOS/GOARCH | per-invocation free vars |
|--------|-----------------------------|--------------------------|
| build  | host-native go binary       | go-ver, target GOOS, target GOARCH (cross via env) |
| test   | must equal host             | go-ver only              |
| lint   | must equal host             | go-ver only              |

`build` cross-compiles via `GOOS`/`GOARCH` env regardless of which host-native go binary runs it. `test`/`lint` execute the binary, so its cell must match the host — a task-level guard rejects mismatched suffixes before running.

### D6 — per-invocation selection via mise exec overlay
Tasks parse `go=`/`goos=`/`goarch=` args and run the command under an explicit version overlay rather than mutating `[tools]`:
```
mise exec tools:go@${GO} [tools:golangci-lint@2.12-go${GO}] -- <cmd>
```
This keeps selection per-invocation and parallel-safe across matrix cells.

### D7 — artifact hosting = GitHub Releases
Raw git is used only for plugin scripts (`.tools/`) and `manifest.json`. All binaries live as Release assets under a deterministic tag scheme: `assets/<tool>/<goos>-<goarch>/go<ver-or-buildgo>/<filename>`.

## Architecture

```
┌────────────────────────── mise-repo (provider) ──────────────────────────┐
│                                                                          │
│  .mise.toml          [tasks] mirror-go, build-tools, publish, manifest-gen │
│  manifest.json      ← source of truth (cells + sha256 + asset urls)      │
│  .tools/            ← vfox plugin hooks (Lua): BackendListVersions / BackendInstall,    │
│                       BackendExecEnv — all read manifest                   │
│  artifacts/         ← local build tree (gitignored), input to publish    │
│  GitHub Releases    ← go×12 mirror + 12 per (tool, version) (27 tools, 31 versions)  │
│                                                                          │
│  build host: 1 linux runner, official `go` plugin bootstraps all tool builds │
└──────────────────────────────────┬───────────────────────────────────────┘
                                   │  mise plugin install tools <url>
                                   ▼
┌──────────────────────── easyp-buf-proxy (consumer) ──────────────────────┐
│  mise.toml                                                               │
│   [tools] "tools:go" = "<cell>"  (suffix-encoded, default host)  │
│           "tools:golangci-lint" = "<cell>"                       │
│   [tasks] build / test / lint  →  mise exec overlay, native guard        │
└──────────────────────────────────────────────────────────────────────────┘
```

## Risks / Trade-offs

- **R1 Release asset size/quota**: ~1.5–2 GB for go+golangci-lint alone; scales ~12 cells per added built tool. GitHub Releases handle this; monitor per-repo storage. Mitigation: keep only current tags, prune old on re-publish.
- **R2 manifest drift**: if `manifest.json` not regenerated, plugin advertises stale cells. Mitigation: `publish` task regenerates and commits manifest atomically with the release.
- **R3 golangci-lint v2 + Go 1.27 beta compatibility**: the linter may fail to build under a beta toolchain. Mitigation: build matrix is best-effort per cell; a failing cell is dropped from `manifest.json` and surfaced in publish logs (not a hard failure of the whole matrix).
- **R4 version-string semver parsing**: vfox semver module may mis-sort heavily-suffixed strings. Mitigation: primary `Available` rows are clean semver; full suffixes are explicit opt-in.
- **R5 host/exec mismatch**: silent at install time (mise will install any cell). Mitigation: consumer task guard (D5) fails fast.

## Alternatives Considered

- **asdf plugin(s)**: rejected — legacy per mise docs, and one-repo-per-tool forces either two clones or fragile `$MISE_PLUGIN_NAME` branching (the earlier "option β"). vfox multi-tool eliminates this.
- **aqua / ubi / github backends**: assume upstream registry shapes; fighting the tool for a self-hosted 2D matrix.
- **Build Go from source**: rejected — bootstrap produces upstream-identical bytes; pure waste.
- **Ship 1 linter binary, build 3 for verification only**: rejected (T3) — user requires 3 shipped, keyed by build-go, for reproducibility.

## Open Questions

- ~~Exact Go 1.27 beta tag to track~~ — RESOLVED: versions are pinned in `versions.toml` (single review knob). `1.27-beta` currently pins `go1.27rc2` (no `go1.27beta<N>` ships upstream yet); swap when a real beta tag appears.
- ~~Exact golangci-lint v2 pin~~ — RESOLVED: `[tools.golangci-lint].version` in `versions.toml` (currently `v2.1.0`).
- Whether to gate `easyp-buf-proxy` GitHub Actions on the new sandbox or keep both — deferred to consumer repo decision.
