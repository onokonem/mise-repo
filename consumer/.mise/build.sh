#!/usr/bin/env sh
# Cross-compile (GOOS/GOARCH free). Uses host-native Go toolchain.
# Args: go=<ver> goos=<os> goarch=<arch>
set -eu
GO=1.26; TARGET_OS=; TARGET_ARCH=
for a in "$@"; do
  case "$a" in
    go=*)    GO="${a#go=}" ;;
    goos=*)  TARGET_OS="${a#goos=}" ;;
    goarch=*) TARGET_ARCH="${a#goarch=}" ;;
  esac
done
# build is the only cross-per-invocation task: env-driven, host-native go binary.
. ./.mise/host.sh
export GOOS="${TARGET_OS:-$HOST_OS}" GOARCH="${TARGET_ARCH:-$HOST_ARCH}" CGO_ENABLED=0
mise exec "tools:go@${GO}" -- go build ./...
echo "built for ${GOOS}/${GOARCH} with go${GO}"
