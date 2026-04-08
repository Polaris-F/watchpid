#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
BINARY_NAME="${BINARY_NAME:-watchpid}"
MODULE_PATH="github.com/Polaris-F/watchpid"

VERSION="${1:-${VERSION:-}}"
if [[ -z "$VERSION" ]]; then
  if git -C "$ROOT_DIR" describe --tags --always --dirty >/dev/null 2>&1; then
    VERSION="$(git -C "$ROOT_DIR" describe --tags --always --dirty)"
  else
    VERSION="dev"
  fi
fi

COMMIT="${COMMIT:-$(git -C "$ROOT_DIR" rev-parse --short HEAD 2>/dev/null || echo unknown)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

DEFAULT_PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
  "windows/arm64"
)

if [[ -n "${PLATFORMS:-}" ]]; then
  read -r -a REQUESTED_PLATFORMS <<< "$PLATFORMS"
else
  REQUESTED_PLATFORMS=("${DEFAULT_PLATFORMS[@]}")
fi

if ! command -v sha256sum >/dev/null 2>&1; then
  echo "sha256sum is required" >&2
  exit 1
fi

SUPPORTED_PLATFORMS="$(go tool dist list)"
BUILD_PLATFORMS=()
for platform in "${REQUESTED_PLATFORMS[@]}"; do
  if grep -qx "$platform" <<< "$SUPPORTED_PLATFORMS"; then
    BUILD_PLATFORMS+=("$platform")
    continue
  fi
  echo "Skipping unsupported platform: $platform" >&2
done

if [[ ${#BUILD_PLATFORMS[@]} -eq 0 ]]; then
  echo "No supported platforms selected" >&2
  exit 1
fi

mkdir -p "$DIST_DIR"
rm -rf "$DIST_DIR/build"
mkdir -p "$DIST_DIR/build"

SHA256_FILE="$DIST_DIR/sha256sums.txt"
: > "$SHA256_FILE"

for platform in "${BUILD_PLATFORMS[@]}"; do
  os="${platform%/*}"
  arch="${platform#*/}"
  ext=""
  archive_path=""

  if [[ "$os" == "windows" ]]; then
    ext=".exe"
  fi

  package_name="${BINARY_NAME}_${VERSION}_${os}_${arch}"
  staging_dir="$DIST_DIR/build/$package_name"
  mkdir -p "$staging_dir"

  env GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build \
      -trimpath \
      -ldflags "-s -w -X ${MODULE_PATH}/internal/buildinfo.Version=${VERSION} -X ${MODULE_PATH}/internal/buildinfo.Commit=${COMMIT} -X ${MODULE_PATH}/internal/buildinfo.Date=${DATE}" \
      -o "$staging_dir/$BINARY_NAME$ext" \
      ./cmd/watchpid

  cp "$ROOT_DIR/README.md" "$ROOT_DIR/README_EN.md" "$staging_dir/"

  if [[ "$os" == "windows" ]]; then
    if ! command -v zip >/dev/null 2>&1; then
      echo "zip is required to package Windows artifacts" >&2
      exit 1
    fi
    (
      cd "$DIST_DIR/build"
      zip -qr "$DIST_DIR/${package_name}.zip" "$package_name"
    )
    archive_path="$DIST_DIR/${package_name}.zip"
  else
    tar -C "$DIST_DIR/build" -czf "$DIST_DIR/${package_name}.tar.gz" "$package_name"
    archive_path="$DIST_DIR/${package_name}.tar.gz"
  fi

  (
    cd "$DIST_DIR"
    sha256sum "$(basename "$archive_path")" >> "$(basename "$SHA256_FILE")"
  )
done

rm -rf "$DIST_DIR/build"

echo "Artifacts written to $DIST_DIR"
