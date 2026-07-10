## Why

`easyp-buf-proxy` needs reproducible build/test/lint across a matrix of 4 os/arch targets and 3 Go toolchain versions (1.25, 1.26, 1.27 beta), including a `golangci-lint` v2 binary built per Go toolchain. Today its CI relies on host-provided `setup-go` with no lint step and no local sandbox; there is no single source of truth for the toolchain binaries. We need a self-hosted, version-pinned "sandbox" tool source so any consumer repo can request a specific matrix cell (e.g. "macos arm64 go1.26 build tools") and get identical binaries locally and in CI.

## What Changes

- Add `mise-repo` as a self-hosted binary provider + builder for the matrix.
- Implement a **mise vfox backend plugin** (multi-tool) providing two tools: `go` and `golangci-lint`.
- Build and serve the full matrix (go + 20+ built tools):
  - `go` × 12 — mirrored from upstream official Go tarballs (3 versions × 4 os/arch).
  - 20+ built Go tools (golangci-lint, staticcheck, gofumpt, ...) × 12 — built locally, cross-compiled, one per `(build-go, os/arch)` cell (3 build-go × 4 os/arch).
- Encode the matrix cell in the **tool version string** via suffix: `@<ver>[-go<buildgo>][-<goos>-<goarch>]`; absent suffix = host auto-detect.
- Publish artifacts as **GitHub Release assets** (raw git refuses >100 MB files; Go tarballs exceed that).
- Add matrix build/publish tasks in `mise-repo/.mise.toml` (Go pipeline under `.mise/`) using the official `go` plugin to bootstrap all tool builds. All tool versions are pinned in `versions.toml` at the repo root.
- Document and template the **consumer configuration** for `easyp-buf-proxy` (`mise.toml` with `[tools]`, `[tasks]` build/test/lint) using per-invocation selection via `mise exec` overlay.
- Enforce native-execution guard: `test`/`lint` tasks reject a suffix that does not match the host (executed binary GOOS/GOARCH must equal host); `build` is the only truly cross-per-invocation task (target via `GOOS`/`GOARCH` env).

## Capabilities

### New Capabilities
- `matrix-tool-distribution`: building, hosting, and exposing the go + multi-tool matrix artifacts through a vfox multi-tool plugin with suffix-encoded version selection.
- `consumer-sandbox-tasks`: the consumer-side `mise.toml` shape — tool registration, suffix selection, and `build`/`test`/`lint` task definitions with per-invocation `mise exec` overlay and native-exec guard.

### Modified Capabilities
<!-- none — no existing specs in openspec/specs/ -->

## Impact

- **mise-repo**: new `mise.toml` (build/publish tasks), vfox plugin (Lua hooks + manifest), `manifest.json` (source of truth for `Available` + asset map), GitHub Releases workflow for all matrix assets.
- **easyp-buf-proxy**: new `mise.toml` (consumer); replaces ad-hoc `setup-go` for local dev, complements (does not remove) existing GitHub Actions workflows.
- **Dependencies**: bootstrap Go toolchain via official mise `go` plugin on the build host; `gh` CLI for release upload; pure-Go `golangci-lint` cross-compiled with `CGO_ENABLED=0`.
- **Supply chain**: mirrored Go tarballs and built linters are unsigned by us; attestation (cosign/SLSA) is out of scope for this change (noted as future work).
