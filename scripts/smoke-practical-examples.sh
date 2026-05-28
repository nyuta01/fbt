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

check_daily_qa_ops() {
  local source_dir="examples/daily_qa_ops"
  local name="daily_qa_ops"
  local project="$tmpdir/$name"
  cp -R "$source_dir" "$project"

  # The checked-in wrapper resolves the source checkout when run in-place.
  # The smoke runs from a copied project, so pass the source checkout explicitly.
  perl -0pi -e 's/(command: bin\/fbt-demo-(?:llm|agent)-runner\n)/$1    env:\n      - FBT_SOURCE_ROOT\n/g' "$project/fs_project.yml"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt parse --project-dir "$project" >"$tmpdir/$name-parse.txt"
  grep -q "Manifest written" "$tmpdir/$name-parse.txt"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt plan --project-dir "$project" --select tag:daily_qa >"$tmpdir/$name-plan.txt"
  grep -q "Plan: 2 selected, 1 run, 0 skipped, 1 blocked" "$tmpdir/$name-plan.txt"
  grep -q "run transform.daily_qa_ops.daily_qa_candidates" "$tmpdir/$name-plan.txt"
  grep -q "blocked transform.daily_qa_ops.promote_manual_update" "$tmpdir/$name-plan.txt"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt build --project-dir "$project" --select daily_qa_candidates >"$tmpdir/$name-build-candidates.txt"
  grep -q "success transform.daily_qa_ops.daily_qa_candidates" "$tmpdir/$name-build-candidates.txt"
  test -f "$project/target/artifacts/qa/latest/faq_candidates.md"
  test -f "$project/target/artifacts/qa/latest/manual_patch_candidates.md"
  test -f "$project/target/artifacts/qa/latest/unresolved_questions.md"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt artifact history faq_candidates --project-dir "$project" >"$tmpdir/$name-history-faq.txt"
  grep -q "confidence: structural" "$tmpdir/$name-history-faq.txt"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt build --project-dir "$project" --select promote_manual_update >"$tmpdir/$name-build-promotion.txt"
  grep -q "success transform.daily_qa_ops.promote_manual_update" "$tmpdir/$name-build-promotion.txt"
  test -f "$project/target/artifacts/manual/latest/manual_update.md"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt artifact show manual_update --project-dir "$project" >"$tmpdir/$name-manual-update.txt"
  grep -q "artifact.daily_qa_ops.manual_update" "$tmpdir/$name-manual-update.txt"
}

check_daily_qa_ops
check_example "examples/incident_response_runbook" "incident_response_runbook"
check_example "examples/support_resolution_manual" "support_resolution_manual"

echo "practical-examples-smoke: ok"
