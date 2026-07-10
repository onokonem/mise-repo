# Matrix Tool Distribution

## Purpose

Define the published tool matrix (mirrored Go tarballs + built golangci-lint), the suffix-encoded cell selection scheme, and the single multi-tool vfox plugin backed by a manifest.

## Requirements


### Requirement: Provide Go toolchain matrix via mirrored upstream tarballs

The repo SHALL publish the Go compiler for the matrix `{1.25, 1.26, 1.27-beta} × {linux, darwin} × {amd64, arm64}` (12 cells). Go artifacts SHALL be mirrored from official upstream Go tarballs and SHALL NOT be built from source. Each cell SHALL be available as a GitHub Release asset.

#### Scenario: Full Go matrix published
- **WHEN** the publish task completes
- **THEN** a GitHub Release contains 12 Go assets, one per `(version, os, arch)` cell
- **AND** each asset is byte-identical to the corresponding official upstream Go tarball

#### Scenario: Go 1.27 beta tracked dynamically
- **WHEN** the `1.27` version is requested
- **THEN** the provider resolves it to the latest available `go1.27beta<N>` upstream tag at build time

### Requirement: Provide golangci-lint v2 matrix built per Go toolchain

The repo SHALL publish `golangci-lint` v2 for the matrix `{1.25, 1.26, 1.27-beta} × {linux, darwin} × {amd64, arm64}` (12 cells), where the "build-go" dimension denotes the Go toolchain used to compile the linter. Linter binaries SHALL be cross-compiled with `CGO_ENABLED=0`. A single Linux build runner SHALL be capable of producing all 12 binaries.

#### Scenario: Linter built under each Go toolchain
- **WHEN** the build matrix runs for build-go `1.26`
- **THEN** four linter binaries are produced: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`
- **AND** each was compiled by invoking `go build` under Go 1.26 with `CGO_ENABLED=0`

#### Scenario: Failing linter cell does not abort the matrix
- **WHEN** golangci-lint fails to build under a given Go toolchain (e.g. a beta incompatibility)
- **THEN** that cell is omitted from the manifest
- **AND** the omission is recorded in publish logs
- **AND** the remaining cells still publish

### Requirement: Suffix-encoded version selection

Tool versions SHALL encode the matrix cell as `@<ver>[-go<buildgo>][-<goos>-<goarch>]`. An omitted `<goos>-<goarch>` suffix SHALL resolve to the host platform at install time. An omitted `-go<buildgo>` on `golangci-lint` SHALL resolve to a documented default build-go.

#### Scenario: Host auto-detect when no os/arch suffix
- **WHEN** a consumer requests `tools:go@1.26` on a `darwin/arm64` host
- **THEN** the `darwin-arm64/go1.26` asset is installed

#### Scenario: Fully pinned cell
- **WHEN** a consumer requests `tools:golangci-lint@2.12-go1.26-darwin-arm64`
- **THEN** the linter asset for `(build-go=1.26, darwin, arm64)` is installed regardless of host

#### Scenario: Latest resolves to a clean default
- **WHEN** a consumer requests `tools:golangci-lint@latest`
- **THEN** the highest `2.x` linter built by the default build-go for the host platform is installed

### Requirement: Single vfox multi-tool plugin

The repo SHALL expose both tools through one mise vfox backend plugin registered as `tools`, addressed via `tools:go` and `tools:golangci-lint`. The plugin SHALL NOT shadow upstream mise tools by tool-id; the executable placed on PATH SHALL retain its bare name (`go`, `golangci-lint`).

#### Scenario: One plugin, two tools
- **WHEN** a consumer runs `mise plugin install tools <repo-url>`
- **THEN** both `tools:go` and `tools:golangci-lint` become installable
- **AND** no upstream mise tool-id is overwritten

### Requirement: Manifest is the single source of truth

A `manifest.json` SHALL enumerate every published cell with its tool, version, build-go, os, arch, asset filename, and sha256. The plugin's `Available` hook SHALL advertise only cells present in the manifest. The plugin's `PreInstall` hook SHALL return the asset URL and sha256 looked up from the manifest. The publish task SHALL regenerate the manifest atomically from the built artifact tree.

#### Scenario: Available reflects published assets only
- **WHEN** a cell is missing its Release asset
- **THEN** `Available` does not advertise that cell

#### Scenario: PreInstall returns checksum
- **WHEN** `PreInstall` runs for an advertised cell
- **THEN** it returns the asset download URL and the sha256 from the manifest
- **AND** mise verifies the downloaded asset against that sha256
