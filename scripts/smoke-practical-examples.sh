#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

check_example() {
  local source_dir="$1"
  local selector="$2"
  local name
  name="$(basename "$source_dir")"
  local project="$tmpdir/$name"
  cp -R "$source_dir" "$project"

  go run ./cmd/fbt parse --project-dir "$project" >"$tmpdir/$name-parse.txt"
  grep -q "Manifest written" "$tmpdir/$name-parse.txt"

  go run ./cmd/fbt plan --project-dir "$project" --select "$selector" >"$tmpdir/$name-plan.txt"
  grep -q "Plan: 1 selected" "$tmpdir/$name-plan.txt"
  grep -q "run transform" "$tmpdir/$name-plan.txt"
}

check_example "examples/incident_response_runbook" "incident_response_runbook"
check_example "examples/support_resolution_manual" "support_resolution_manual"

echo "practical-examples-smoke: ok"
