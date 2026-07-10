# tools consumer template

Copy these files into a Go module repo (e.g. `easyp-buf-proxy`) to get build /
test / lint against the self-hosted toolchain matrix with **no system Go
required**.

## Files

| file            | purpose                                                       |
|-----------------|---------------------------------------------------------------|
| `mise.toml`     | tool pins + `build` / `test` / `lint` tasks (per-invocation)  |
| `.golangci.yml` | golangci-lint v2 config (loaded by the lint task)             |
| `.mise/host.sh` | host GOOS/GOARCH detection for the native-execution guard     |

## One-time setup

```sh
mise plugin install tools \
  "git::https://github.com/onokonem/mise-repo.git//.tools"
mise install        # installs host-matching go + golangci-lint
```

The `git::<url>//.tools` form installs only the plugin subdirectory (the repo
root holds provider tooling, not the plugin).

## Per-invocation usage

```sh
mise run test  -- go=1.25                          # host cell, Go 1.25, -race
mise run build -- go=1.26 goos=linux goarch=arm64  # cross-compile (host go binary)
mise run lint  -- go=1.26                          # host cell, golangci-lint v2
```

### Native-execution guard

`test` and `lint` **execute** a tool binary, so the cell's GOOS/GOARCH must
equal the host. They fail fast with an explanatory message if you pass a
non-host `goos=`/`goarch=`. `build` is exempt — it cross-compiles via
`GOOS`/`GOARCH` env using a host-native Go binary.

### `-race` prerequisite

`test` runs `go test -race`, which needs CGO and a host C compiler
(`gcc` on Linux, `clang` via Xcode command-line tools on macOS). Install it if
the race detector reports a missing CGO toolchain.

## Available cells

```sh
mise ls tools:go               # 1.25, 1.26, 1.27-beta (+ pinned cells)
mise ls tools:golangci-lint    # 2.12, 2.12-go1.26, ... (+ pinned cells)
```

Cells absent from this list were skipped at build time (e.g. a linter that
failed under a beta toolchain). Requesting one fails cleanly with a message.
