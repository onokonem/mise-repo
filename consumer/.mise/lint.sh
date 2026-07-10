#!/usr/bin/env sh
# Run golangci-lint against .golangci.yml. Cell MUST be host.
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
  echo "lint executes golangci-lint, so the cell must match the host:" >&2
  echo "  requested goos=$REQ_OS, host=$HOST_OS." >&2
  exit 1
fi
if [ -n "$REQ_ARCH" ] && [ "$REQ_ARCH" != "$HOST_ARCH" ]; then
  echo "lint executes golangci-lint, so the cell must match the host:" >&2
  echo "  requested goarch=$REQ_ARCH, host=$HOST_ARCH." >&2
  exit 1
fi
# golangci-lint shells out to `go` for package loading -> overlay BOTH.
mise exec "tools:go@${GO}" "tools:golangci-lint@2.12-go${GO}" -- golangci-lint run
