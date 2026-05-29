#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

project="$tmpdir/daily_qa_ops"
cp -R examples/daily_qa_ops "$project"

# The copied project still uses repository-local demo runner wrappers.
perl -0pi -e 's/(command: bin\/fbt-demo-(?:llm|agent)-runner\n)/$1    env:\n      - FBT_SOURCE_ROOT\n/g' "$project/fs_project.yml"

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt plan --project-dir "$project" --select tag:daily_qa >"$tmpdir/day1-plan.txt"
grep -q "selected  2" "$tmpdir/day1-plan.txt"
grep -q "RUN     daily_qa_candidates" "$tmpdir/day1-plan.txt"
grep -q "RUN     promote_manual_update" "$tmpdir/day1-plan.txt"

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt build --project-dir "$project" --select tag:daily_qa >"$tmpdir/day1-build.txt"
grep -q "SUCCESS daily_qa_candidates" "$tmpdir/day1-build.txt"
grep -q "SUCCESS promote_manual_update" "$tmpdir/day1-build.txt"
test -f "$project/target/artifacts/qa/latest/faq_candidates.md"
test -f "$project/target/artifacts/qa/latest/manual_patch_candidates.md"
test -f "$project/target/artifacts/qa/latest/unresolved_questions.md"
test -f "$project/target/artifacts/manual/latest/manual_update.md"

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt artifact explain manual_update --project-dir "$project" >"$tmpdir/day1-explain.txt"
grep -Eq "input +manual_patch_candidates" "$tmpdir/day1-explain.txt"
grep -Eq "input +unresolved_questions" "$tmpdir/day1-explain.txt"
grep -Eq "input +reference\\.current_manual" "$tmpdir/day1-explain.txt"

cat >"$project/data/qa/inbox/questions/Q-1044.md" <<'MD'
# Q-1044: Admin export timezone

Customer asks whether scheduled admin exports use the workspace timezone or UTC.
MD

cat >"$project/data/qa/inbox/answers/A-1044.md" <<'MD'
# A-1044

Scheduled exports use the workspace timezone unless the export job explicitly
sets UTC in the admin settings.
MD

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt plan --project-dir "$project" --select daily_qa_candidates >"$tmpdir/day2-plan.txt"
grep -q "selected  1" "$tmpdir/day2-plan.txt"
grep -q "because  source descriptor changed" "$tmpdir/day2-plan.txt"

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt build --project-dir "$project" --select daily_qa_candidates >"$tmpdir/day2-build.txt"
grep -q "SUCCESS daily_qa_candidates" "$tmpdir/day2-build.txt"

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt artifact history faq_candidates --project-dir "$project" >"$tmpdir/day2-history.txt"
grep -q "Artifact: faq_candidates" "$tmpdir/day2-history.txt"
grep -q "Status      current" "$tmpdir/day2-history.txt"

echo "daily-ops-smoke: ok"
