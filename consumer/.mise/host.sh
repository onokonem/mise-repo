# host.sh — detect the host Go (GOOS, GOARCH) cell without requiring Go.
# Sourced by the build/test/lint tasks to drive the native-execution guard.
# Exports HOST_OS and HOST_ARCH.
HOST_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$(uname -m)" in
  x86_64)        HOST_ARCH="amd64" ;;
  aarch64|arm64) HOST_ARCH="arm64" ;;
  *)             HOST_ARCH="$(uname -m)" ;;
esac
export HOST_OS HOST_ARCH
