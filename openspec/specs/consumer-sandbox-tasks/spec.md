# Consumer Sandbox Tasks

## Purpose

Define how a consumer Go module registers the mise-repo sandbox plugin and drives per-invocation build/test/lint cells via mise tasks.

## Requirements


### Requirement: Consumer registers the sandbox plugin

The consumer repo (`easyp-buf-proxy`) SHALL declare the sandbox as its tool source by registering the `tools` vfox plugin and pinning both `tools:go` and `tools:golangci-lint` under `[tools]` in its `mise.toml`. Default pins SHALL resolve to the host platform so a plain `mise install` works on any supported host.

#### Scenario: Default install works on any host
- **WHEN** a developer runs `mise install` in the consumer repo on a supported host
- **THEN** a host-matching Go toolchain and golangci-lint are installed with no host system Go required

### Requirement: Per-invocation cell selection

The consumer SHALL select a matrix cell per task invocation via arguments (`go=`, `goos=`, `goarch=`) applied through a `mise exec` version overlay, without mutating `[tools]`. Selection SHALL NOT require reinstalling tools or editing `mise.toml`.

#### Scenario: Select Go version per run
- **WHEN** a developer runs `mise run test -- go=1.25`
- **THEN** the test task executes under Go 1.25 for the host platform
- **AND** no other cell is affected

#### Scenario: Cross-build target per run
- **WHEN** a developer runs `mise run build -- go=1.26 goos=linux goarch=arm64` from a `darwin/arm64` host
- **THEN** the build task produces a `linux/arm64` binary using the host-native Go 1.26 toolchain

### Requirement: Native-execution guard for test and lint

The `test` and `lint` tasks execute tool binaries, so the selected cell's GOOS/GOARCH SHALL equal the host. The tasks SHALL fail fast with an explanatory message when a requested suffix does not match the host. The `build` task SHALL be exempt because it cross-compiles via environment variables using a host-native Go binary.

#### Scenario: Mismatched lint suffix rejected
- **WHEN** a developer runs `mise run lint -- go=1.26 goos=linux goarch=arm64` on a `darwin/arm64` host
- **THEN** the task fails before invoking golangci-lint
- **AND** the error explains that lint requires a host-matching cell

#### Scenario: Cross build allowed despite host mismatch
- **WHEN** a developer runs `mise run build -- go=1.26 goos=linux goarch=amd64` on a `darwin/arm64` host
- **THEN** the task succeeds and emits a `linux/amd64` binary

### Requirement: Build, test, and lint task definitions

The consumer `mise.toml` SHALL define `build`, `test`, and `lint` tasks. `test` SHALL run `go test ./... -race -count=1`. `lint` SHALL run `golangci-lint run` against the consumer's existing `.golangci.yml`. `build` SHALL run `go build ./...`. Each task SHALL apply the per-invocation `mise exec` overlay and the native-execution guard where applicable.

#### Scenario: Lint uses consumer config
- **WHEN** a developer runs `mise run lint`
- **THEN** golangci-lint executes with the consumer repo's `.golangci.yml`
- **AND** exits non-zero on any finding

#### Scenario: Test runs with race detector
- **WHEN** a developer runs `mise run test`
- **THEN** `go test` is invoked with `-race -count=1` across all packages
