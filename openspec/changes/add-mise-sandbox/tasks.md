# Tasks — add-mise-sandbox

> Implementation status: all **authoring** tasks (sections 1–7) are complete and
> the plugin was verified to load end-to-end (`mise ls-remote` returns derived
> versions, absent-cell requests fail with the explanatory error, `openspec
> validate` passes). Tasks in **section 8** require executing the published
> pipeline (real GitHub Release + CI linux runner + the consumer repo) and are
> left unchecked — see the apply-session summary.
>
> Layout note: provider config is `.mise.toml`; the pipeline is **Go**
> (`.mise/cmd/*` + `.mise/internal/provider`, module root `.mise`), run via
> `go run` from `.mise` with Go installed from mise's standard source
> (`mise use -g go@latest`). **All tool versions are pinned** in `versions.toml`
> at the repo root: `[tools.go]` (mirrored Go tags) + one `[tools.<name>]`
> subtable per built tool (`version`, `path`, `binary`, optional `key`) — 27 tools, 31 versions incl. legacy (golangci-lint v1, mockery v2, buf, protoc-gen-go-grpc) from
> the prior Dockerfile provider, minus private yadro.* + non-Go). The
> `build-tools` task `go install`s every built tool across the matrix. No
> version is resolved online at build time; the go.dev index is fetched only to
> verify upstream Go sha256.
>
> Mechanism note (load-bearing, discovered from mise source): a multi-tool
> plugin is a vfox **backend** plugin, detected by `metadata.lua` +
> `hooks/backend_install.lua`. The hooks implemented are therefore
> `BackendListVersions` / `BackendInstall` / `BackendExecEnv` (each receives
> `ctx.tool`), which are the multi-tool equivalents of the single-tool
> `Available` / `PreInstall` / `EnvKeys` named in the original task text.

## 1. Provider repo scaffolding (mise-repo)

- [x] 1.1 Add `.gitignore` for `artifacts/` and local build output
- [x] 1.2 Define `manifest.json` schema with fields: `tool`, `version` (clean key, e.g. `1.26`), `resolved_tag` (concrete upstream/built tag, e.g. `go1.27beta3`, `v2.1.6`), `buildgo`, `os`, `arch`, `asset` (filename), `asset_url` (full Release URL), `sha256`
- [x] 1.3 Create `.tools/` vfox multi-tool plugin: `metadata.lua` declaring BOTH tools (`go`, `golangci-lint`) for the `tools` backend + hooks dir
- [x] 1.4 Bake Release base URL (`https://github.com/<owner>/mise-repo/releases/download/<tag>`) into plugin config (`<plugin>/.tools/config.lua`), read by `PreInstall` to build `asset_url` when manifest omits it

## 2. Go mirror pipeline

- [x] 2.1 Add `mirror-go` mise task: read fixed tags for `{1.25, 1.26, 1.27-beta}` from `versions.toml` (no dynamic resolution; `1.27-beta` pinned to `go1.27rc2`, adjustable) and record each as the manifest `resolved_tag`
- [x] 2.2 Download 12 official Go tarballs into `artifacts/go/<goos>-<goarch>/go<ver>/`
- [x] 2.3 Compute sha256 per asset; verify each against upstream official Go CHECKSUMS file before serving (byte-identity guard)
- [x] 2.4 Feed `resolved_tag` + sha256 into manifest generation

## 3. Built-tools build pipeline (27 tools, 31 versions)

- [x] 3.1 Add `build-tools` mise task: iterate every (tool, version) in `versions.toml` (`[[tools.<name>]]`) × buildgo ∈ {1.25, 1.26, 1.27-beta} × {linux, darwin} × {amd64, arm64}
- [x] 3.2 Per (tool, buildgo, os, arch) cell: `mise exec go@<buildgo> -- env GOOS GOARCH CGO_ENABLED=0 GOBIN=<dir> go install -trimpath -ldflags="-s -w" <path>@<version>` (version recorded as `resolved_tag`; method matches the prior Dockerfile provider in `Tools.txt`)
- [x] 3.3 Emit each binary tarball to `artifacts/<tool>/<goos>-<goarch>/go<buildgo>/`
- [x] 3.4 Make a failing cell non-fatal: log + skip, continue matrix, drop from manifest

## 4. Publish + manifest

- [x] 4.1 `manifest-gen` task: scan `artifacts/`, emit `manifest.json` with asset paths + `asset_url` + `resolved_tag` + sha256; omit skipped cells
- [x] 4.2 `publish` task: define Release **tag scheme** (e.g. `sandbox-<YYYYMMDD>-<short-sha>`); upload all non-skipped assets (≤ 24) via `gh release upload`
- [x] 4.3 Order: upload all assets first → regenerate manifest from the created Release → commit manifest; on partial upload, abort the Release and leave manifest uncommitted (no drift)
- [x] 4.4 Prune task: delete prior sandbox Release tags on re-publish (R1 bloat mitigation)

## 5. vfox plugin hooks

- [x] 5.1 `Available(tool)`: read committed `manifest.json` (shipped inside plugin clone), emit clean default rows + full suffix rows; normalize version keys (`1.27-beta` ↔ `go1.27beta<N>` via `resolved_tag`) so semver sort stays stable
- [x] 5.2 `PreInstall(tool, version, ctx)`: parse suffix, resolve host/default, return `{url, sha256}` from manifest; if requested cell absent from manifest, fail with explanatory error (covers skipped-cell request)
- [x] 5.3 `EnvKeys(tool)`: export GOROOT/GOPATH for `go` (derive GOROOT from extracted tarball, verify `go env GOROOT` resolves); none for `golangci-lint`
- [x] 5.4 Verify plugin loads: `mise plugin install tools <local-path>` + `mise ls tools:go`; confirm `manifest.json` is present in cloned plugin dir and base URL resolves

## 6. Consumer config (easyp-buf-proxy)

- [x] 6.1 Add consumer `mise.toml`: register `tools`, pin default host cells under `[tools]`
- [x] 6.2 `[tasks.build]`: parse `go/goos/goarch`, overlay `go@<go>` (host-native), cross-compile via `GOOS`/`GOARCH`
- [x] 6.3 `[tasks.test]`: parse `go`, native-guard, `mise exec go@<go> -- go test ./... -race -count=1`; document host C-compiler (gcc/clang) prereq for `-race`
- [x] 6.4 `[tasks.lint]`: parse `go`, native-guard, overlay BOTH `go@<go>` AND linter: `mise exec go@<go> tools:golangci-lint@2.12-go<go> -- golangci-lint run` (linter shells to `go` for package loading)
- [x] 6.5 Native-guard helper: fail fast when test/lint resolved cell ≠ host
- [x] 6.6 Verify consumer `.golangci.yml` exists (lint scenario depends on it)

## 7. CI workflow

- [x] 7.1 Add GitHub Actions workflow on `mise-repo`: 1 linux runner, install Go (latest stable) via mise, run `mirror-go` → `build-tools` → `publish` on workflow_dispatch/weekly

## 8. Validation

- [ ] 8.1 Provider: build full matrix on one linux runner, confirm all non-skipped assets published (≤ 24; exact 24 only if no cell skipped)
- [ ] 8.2 Provider: confirm manifest ↔ Release assets in sync (no orphan/missing cells)
- [ ] 8.3 Provider: confirm each Go asset sha256 matches upstream CHECKSUMS
- [ ] 8.4 Consumer: `mise install` works clean on darwin/arm64 host (default host cells), no host Go required
- [ ] 8.5 Consumer: `mise run build -- go=1.26 goos=linux goarch=arm64` produces linux/arm64 binary
- [ ] 8.6 Consumer: `mise run test -- go=1.25` and `mise run lint -- go=1.26` pass (lint overlay includes `go@<go>`)
- [ ] 8.7 Consumer: mismatched lint suffix fails with explanatory error
- [x] 8.8 Consumer: requesting a cell absent from manifest fails cleanly (skipped-cell path)
- [x] 8.9 `openspec validate add-mise-sandbox` passes
