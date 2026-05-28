#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

dist_dir="$ROOT_DIR/dist"
mkdir -p "$dist_dir"

VERSION="${VERSION:-0.1.0}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
LDFLAGS="-s -w -X github.com/nyuta01/fbt/internal/version.Version=${VERSION} -X github.com/nyuta01/fbt/internal/version.Commit=${COMMIT} -X github.com/nyuta01/fbt/internal/version.BuildDate=${BUILD_DATE}"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"
binary="$dist_dir/fbt_${GOOS}_${GOARCH}"

go build -trimpath -ldflags "$LDFLAGS" -o "$binary" ./cmd/fbt
"$binary" version >"$dist_dir/version.txt"
grep -q "^fbt ${VERSION}$" "$dist_dir/version.txt"
"$binary" version --json >"$dist_dir/version.json"
grep -q "\"version\": \"${VERSION}\"" "$dist_dir/version.json"
grep -q "\"commit\": \"${COMMIT}\"" "$dist_dir/version.json"
grep -q "\"build_date\": \"${BUILD_DATE}\"" "$dist_dir/version.json"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
"$binary" init "$tmpdir/blank" --template blank >"$tmpdir/init.txt"
"$binary" plan --project-dir "$tmpdir/blank" >"$tmpdir/plan.txt"
grep -q "Plan:" "$tmpdir/plan.txt"
test -f "$tmpdir/blank/.fbt/state/manifest.json"

echo "dist-check: ok"
