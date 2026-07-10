#!/usr/bin/env sh
# Run tests with -race. Cell MUST be host (native guard enforced).
# Args: go=<ver> [goos=<os>] [goarch=<arch>]
set -eu
GO=1.26; REQ_OS=; REQ_ARCH=
for a in "$@"; do
  case "$a" in
    go=*) GO="${a#go=}" ;;
    goos=*) REQ_OS="${a#goos=}" ;;
    goarch=*) REQ_ARCH="${a#goarch=}" ;;
  esac
done
. ./.mise/host.sh
if [ -n "$REQ_OS" ] && [ "$REQ_OS" != "$HOST_OS" ]; then
  echo "test executes the Go binary, so the cell must match the host:" >&2
  echo "  requested goos=$REQ_OS, host=$HOST_OS. Use 'mise run build' for cross." >&2
  exit 1
fi
if [ -n "$REQ_ARCH" ] && [ "$REQ_ARCH" != "$HOST_ARCH" ]; then
  echo "test executes the Go binary, so the cell must match the host:" >&2
  echo "  requested goarch=$REQ_ARCH, host=$HOST_ARCH. Use 'mise run build' for cross." >&2
  exit 1
fi
# -race needs a host C compiler (gcc on Linux, clang on macOS). Install it if
# the race detector complains about missing CGO.
mise exec "tools:go@${GO}" -- go test ./... -race -count=1
