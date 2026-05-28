#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

go run ./cmd/fbt --help >"$tmpdir/help.txt"
grep -q "file build tool" "$tmpdir/help.txt"
grep -q "Planned commands" "$tmpdir/help.txt"

go run ./cmd/fbt version >"$tmpdir/version.txt"
grep -q "^fbt 0.0.0-dev$" "$tmpdir/version.txt"

if go run ./cmd/fbt build >"$tmpdir/build.out" 2>"$tmpdir/build.err"; then
  echo "expected fbt build to be a planned-but-unimplemented command" >&2
  exit 1
fi
grep -q "not implemented yet" "$tmpdir/build.err"

echo "cli-smoke: ok"

