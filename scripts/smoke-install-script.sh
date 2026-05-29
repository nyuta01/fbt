#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

VERSION="${VERSION:-$(awk '/^VERSION \?= / {print $3}' Makefile)}"
COMMIT="${COMMIT:-installsmoke}"
BUILD_DATE="${BUILD_DATE:-2026-01-01T00:00:00Z}"
GOOS_VALUE="$(go env GOOS)"
GOARCH_VALUE="$(go env GOARCH)"

case "$GOOS_VALUE" in
  darwin|linux|windows)
    ;;
  *)
    echo "install-script-smoke: unsupported host GOOS $GOOS_VALUE" >&2
    exit 1
    ;;
esac

case "$GOARCH_VALUE" in
  amd64|arm64)
    ;;
  *)
    echo "install-script-smoke: unsupported host GOARCH $GOARCH_VALUE" >&2
    exit 1
    ;;
esac

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

release_dir="$tmpdir/release"
work_dir="$tmpdir/work"
install_dir="$tmpdir/bin"
mkdir -p "$release_dir" "$work_dir" "$install_dir"

binary_name="fbt"
archive_name="fbt_${VERSION}_${GOOS_VALUE}_${GOARCH_VALUE}.tar.gz"
if [[ "$GOOS_VALUE" == "windows" ]]; then
  binary_name="fbt.exe"
  archive_name="fbt_${VERSION}_${GOOS_VALUE}_${GOARCH_VALUE}.zip"
fi

ldflags="-s -w -X github.com/nyuta01/fbt/internal/version.Version=${VERSION} -X github.com/nyuta01/fbt/internal/version.Commit=${COMMIT} -X github.com/nyuta01/fbt/internal/version.BuildDate=${BUILD_DATE}"
GOOS="$GOOS_VALUE" GOARCH="$GOARCH_VALUE" go build -trimpath -ldflags "$ldflags" -o "$work_dir/$binary_name" ./cmd/fbt

if [[ "$GOOS_VALUE" == "windows" ]]; then
  command -v zip >/dev/null || {
    echo "zip is required for install-script-smoke on Windows" >&2
    exit 1
  }
  (cd "$work_dir" && zip -q "$release_dir/$archive_name" "$binary_name")
else
  tar -C "$work_dir" -czf "$release_dir/$archive_name" "$binary_name"
fi

(
  cd "$release_dir"
  if command -v shasum >/dev/null; then
    shasum -a 256 "$archive_name" >SHA256SUMS
  else
    sha256sum "$archive_name" >SHA256SUMS
  fi
)

FBT_VERSION="v$VERSION" \
FBT_INSTALL_DIR="$install_dir" \
FBT_DOWNLOAD_BASE_URL="file://$release_dir" \
  sh install.sh >"$tmpdir/install.txt"

"$install_dir/fbt" version >"$tmpdir/version.txt"
grep -q "^fbt ${VERSION}$" "$tmpdir/version.txt"
grep -q "installed fbt v${VERSION}" "$tmpdir/install.txt"

echo "install-script-smoke: ok"
