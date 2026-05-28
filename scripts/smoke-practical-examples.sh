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

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt plan --project-dir "$project" --select daily_qa_candidates >"$tmpdir/$name-plan-clean.txt"
  grep -q "Plan: 1 selected, 0 run, 1 skipped, 0 blocked" "$tmpdir/$name-plan-clean.txt"

  cat >"$project/data/qa/inbox/questions/Q-1044.md" <<'EOF'
# Q-1044: Admin export timezone

Customer asks whether scheduled admin exports use the workspace timezone or UTC.
EOF

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt plan --project-dir "$project" --select daily_qa_candidates >"$tmpdir/$name-plan-new-window.txt"
  grep -q "Plan: 1 selected, 1 run, 0 skipped, 0 blocked" "$tmpdir/$name-plan-new-window.txt"
  grep -q "reason: source descriptor changed" "$tmpdir/$name-plan-new-window.txt"
  grep -q "next: fbt build --select daily_qa_candidates" "$tmpdir/$name-plan-new-window.txt"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt build --project-dir "$project" --select promote_manual_update >"$tmpdir/$name-build-promotion.txt"
  grep -q "success transform.daily_qa_ops.promote_manual_update" "$tmpdir/$name-build-promotion.txt"
  test -f "$project/target/artifacts/manual/latest/manual_update.md"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt artifact show manual_update --project-dir "$project" >"$tmpdir/$name-manual-update.txt"
  grep -q "artifact.daily_qa_ops.manual_update" "$tmpdir/$name-manual-update.txt"
}

check_markdown_toolchain() {
  local source_dir="examples/markdown_toolchain"
  local name="markdown_toolchain"
  local project="$tmpdir/$name"
  cp -R "$source_dir" "$project"

  perl -0pi -e 's/(command: bin\/fbt-command-runner\n)/$1    env:\n      - FBT_SOURCE_ROOT\n/g' "$project/fs_project.yml"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt plan --project-dir "$project" --select tag:document_toolchain >"$tmpdir/$name-plan.txt"
  grep -q "Plan: 2 selected, 1 run, 0 skipped, 1 blocked" "$tmpdir/$name-plan.txt"
  grep -q "run transform.markdown_toolchain.remark_markdown" "$tmpdir/$name-plan.txt"
  grep -q "blocked transform.markdown_toolchain.pandoc_handbook" "$tmpdir/$name-plan.txt"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt build --project-dir "$project" --select remark_markdown >"$tmpdir/$name-build-remark.txt"
  grep -q "success transform.markdown_toolchain.remark_markdown" "$tmpdir/$name-build-remark.txt"
  test -f "$project/target/artifacts/markdown/normalized/handbook.md"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt build --project-dir "$project" --select pandoc_handbook >"$tmpdir/$name-build-pandoc.txt"
  grep -q "success transform.markdown_toolchain.pandoc_handbook" "$tmpdir/$name-build-pandoc.txt"
  test -f "$project/target/artifacts/documents/handbook.pdf"

  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt artifact show handbook_pdf --project-dir "$project" >"$tmpdir/$name-handbook.txt"
  grep -q "type: fbt.artifact.pdf_document.v1" "$tmpdir/$name-handbook.txt"
}

check_daily_qa_ops
check_markdown_toolchain
check_example "examples/incident_response_runbook" "incident_response_runbook"
check_example "examples/support_resolution_manual" "support_resolution_manual"

echo "practical-examples-smoke: ok"
