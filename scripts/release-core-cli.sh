#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ $# -ne 1 ]]; then
  echo "usage: scripts/release-core-cli.sh vX.Y.Z" >&2
  exit 2
fi

tag="$1"
version="${tag#v}"
if [[ "$tag" == "$version" || ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "release tag must look like vX.Y.Z" >&2
  exit 2
fi

command -v zip >/dev/null || {
  echo "zip is required to build Windows release archives" >&2
  exit 2
}

commit="${COMMIT:-$(git rev-parse --short HEAD)}"
build_date="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
release_dir="$ROOT_DIR/dist/release/$tag"
work_dir="$release_dir/work"

rm -rf "$release_dir"
mkdir -p "$work_dir"

ldflags="-s -w -X github.com/nyuta01/fbt/internal/version.Version=${version} -X github.com/nyuta01/fbt/internal/version.Commit=${commit} -X github.com/nyuta01/fbt/internal/version.BuildDate=${build_date}"

platforms=(
  "darwin amd64 tar.gz"
  "darwin arm64 tar.gz"
  "linux amd64 tar.gz"
  "linux arm64 tar.gz"
  "windows amd64 zip"
  "windows arm64 zip"
)

for platform in "${platforms[@]}"; do
  read -r goos goarch archive_type <<<"$platform"
  target_dir="$work_dir/${goos}_${goarch}"
  mkdir -p "$target_dir"

  binary_name="fbt"
  if [[ "$goos" == "windows" ]]; then
    binary_name="fbt.exe"
  fi

  GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "$ldflags" -o "$target_dir/$binary_name" ./cmd/fbt

  case "$archive_type" in
    tar.gz)
      tar -C "$target_dir" -czf "$release_dir/fbt_${version}_${goos}_${goarch}.tar.gz" "$binary_name"
      ;;
    zip)
      (cd "$target_dir" && zip -q "$release_dir/fbt_${version}_${goos}_${goarch}.zip" "$binary_name")
      ;;
  esac
done

"$work_dir/darwin_arm64/fbt" version --json >"$release_dir/version.json"
grep -q "\"version\": \"${version}\"" "$release_dir/version.json"
grep -q "\"commit\": \"${commit}\"" "$release_dir/version.json"
grep -q "\"build_date\": \"${build_date}\"" "$release_dir/version.json"

(
  cd "$release_dir"
  shasum -a 256 fbt_* version.json >SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)

rm -rf "$work_dir"
echo "release-core-cli: $release_dir"
