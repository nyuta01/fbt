#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

project="$tmpdir/semantic_eval_boundary"
cp -R examples/semantic_eval_boundary "$project"

perl -0pi -e 's/(command: bin\/fbt-command-runner\n)/$1    env:\n      - FBT_SOURCE_ROOT\n/g' "$project/fs_project.yml"

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt plan --project-dir "$project" --select tag:quality_boundary >"$tmpdir/plan.txt"
grep -q "selected  2" "$tmpdir/plan.txt"
grep -q "RUN     manual_update" "$tmpdir/plan.txt"
grep -q "RUN     evidence_quality_report" "$tmpdir/plan.txt"

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt build --project-dir "$project" --select tag:quality_boundary >"$tmpdir/build.txt"
grep -q "SUCCESS manual_update" "$tmpdir/build.txt"
grep -q "SUCCESS evidence_quality_report" "$tmpdir/build.txt"

test -f "$project/target/artifacts/manual/manual_update.md"
test -f "$project/target/artifacts/quality/evidence_quality_report.md"
grep -q "Result: pass" "$project/target/artifacts/quality/evidence_quality_report.md"
grep -q "external command runner" "$project/target/artifacts/quality/evidence_quality_report.md"

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt artifact explain evidence_quality_report --project-dir "$project" >"$tmpdir/explain.txt"
grep -Eq "input +manual_update" "$tmpdir/explain.txt"
grep -Eq "input +source\\.incident_evidence" "$tmpdir/explain.txt"
grep -Eq "asset +evidence_quality_rubric" "$tmpdir/explain.txt"
grep -Eq "runner +local\\.command" "$tmpdir/explain.txt"

echo "semantic-eval-boundary-smoke: ok"
