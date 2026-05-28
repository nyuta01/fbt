#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

dist_dir="$ROOT_DIR/dist"
mkdir -p "$dist_dir"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"
binary="$dist_dir/fbt_${GOOS}_${GOARCH}"

go build -trimpath -ldflags "-s -w" -o "$binary" ./cmd/fbt
"$binary" version >"$dist_dir/version.txt"
grep -q '^fbt 0.0.0-dev$' "$dist_dir/version.txt"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
"$binary" init "$tmpdir/blank" --template blank >"$tmpdir/init.txt"
"$binary" parse --project-dir "$tmpdir/blank" >"$tmpdir/parse.txt"
grep -q "Manifest written" "$tmpdir/parse.txt"

echo "dist-check: ok"
