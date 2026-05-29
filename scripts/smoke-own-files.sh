#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

project="$tmpdir/my_support_manual"

go run ./cmd/fbt init "$project" --template support >"$tmpdir/init.txt"
grep -q "Initialized support project" "$tmpdir/init.txt"

rm -f "$project/data/support/tickets/"*.jsonl
cat >"$project/data/support/tickets/2026-05-29-login.jsonl" <<'JSONL'
{"id":"MY-101","summary":"Admin cannot receive password reset email after company domain change","impact":"Workspace admin blocked from console","resolution_status":"resolved"}
JSONL
cat >"$project/data/support/tickets/2026-05-29-billing.jsonl" <<'JSONL'
{"id":"MY-102","summary":"Customer asks why seat count changed after SSO group sync","impact":"Unexpected invoice estimate","resolution_status":"resolved"}
JSONL
cat >"$project/assets/support_style_guide.md" <<'MD'
# Support Style Guide

- Separate facts from assumptions.
- Include the customer impact and next action.
- Keep generated notes short enough to review in a pull request.
MD

go run ./cmd/fbt doctor --project-dir "$project" >"$tmpdir/doctor.txt"
grep -q "Doctor: ok" "$tmpdir/doctor.txt"

go run ./cmd/fbt plan --project-dir "$project" --select case_summaries >"$tmpdir/plan.txt"
grep -q "selected  1" "$tmpdir/plan.txt"
grep -q "RUN     case_summaries" "$tmpdir/plan.txt"
grep -q "because  output missing" "$tmpdir/plan.txt"

go run ./cmd/fbt build --project-dir "$project" --select case_summaries >"$tmpdir/build.txt"
grep -q "SUCCESS case_summaries" "$tmpdir/build.txt"
test -f "$project/target/artifacts/support/case_summaries/index.md"

go run ./cmd/fbt artifact explain case_summaries --project-dir "$project" >"$tmpdir/explain.txt"
grep -Eq "input +support\\.raw_tickets" "$tmpdir/explain.txt"
grep -Fq "path=data/support/tickets/*.jsonl" "$tmpdir/explain.txt"
grep -Eq "asset +support_style_guide" "$tmpdir/explain.txt"
grep -Eq "runner +demo\\.llm" "$tmpdir/explain.txt"
grep -q "target/artifacts/support/case_summaries" "$tmpdir/explain.txt"

cat >"$project/data/support/tickets/2026-05-30-export.jsonl" <<'JSONL'
{"id":"MY-103","summary":"Customer asks whether scheduled exports use UTC or workspace timezone","impact":"Admin unsure how to communicate report timing","resolution_status":"open"}
JSONL

go run ./cmd/fbt plan --project-dir "$project" --select case_summaries >"$tmpdir/plan-new-source.txt"
grep -q "selected  1" "$tmpdir/plan-new-source.txt"
grep -q "because  source descriptor changed" "$tmpdir/plan-new-source.txt"

echo "own-files-smoke: ok"
