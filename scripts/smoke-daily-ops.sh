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

FBT_SOURCE_ROOT="$ROOT_DIR" \
  FBT_BIN="go run ./cmd/fbt" \
  FBT_PROJECT_DIR="$project" \
  FBT_RUN_ID="smoke-daily-ops" \
  "$project/ops/run-daily.sh" >"$tmpdir/ops-run.txt"
grep -q "fbt daily support knowledge run complete" "$tmpdir/ops-run.txt"
test -f "$project/target/ops/runs/smoke-daily-ops/summary.md"
test -f "$project/target/ops/runs/smoke-daily-ops/source-window.txt"
test -f "$project/target/ops/runs/smoke-daily-ops/doctor.txt"
test -f "$project/target/ops/runs/smoke-daily-ops/plan.txt"
test -f "$project/target/ops/runs/smoke-daily-ops/build.txt"
test -f "$project/target/ops/runs/smoke-daily-ops/manual_update-explain.txt"
test -f "$project/target/ops/runs/smoke-daily-ops/retention.txt"
test -f "$project/target/ops/runs/smoke-daily-ops/openlineage.ndjson"
test -f "$project/target/ops/runs/smoke-daily-ops/otel.json"
test -f "$project/target/ops/runs/smoke-daily-ops/quality-gates.txt"
test -f "$project/target/ops/runs/smoke-daily-ops/quality-gates.json"
test -f "$project/target/ops/runs/smoke-daily-ops/archive.txt"
test -f "$project/target/ops/runs/smoke-daily-ops/publish-handoff.txt"
test -f "$project/target/ops/archives/smoke-daily-ops/fbt-evidence.tar.gz"
test -f "$project/target/ops/archives/smoke-daily-ops/archive-manifest.json"
test -f "$project/target/ops/publish/smoke-daily-ops/publish-manifest.json"
test -f "$project/target/ops/publish/smoke-daily-ops/pr-body.md"
test -f "$project/target/ops/publish/smoke-daily-ops/notification.md"
test -f "$project/target/ops/latest/summary.md"
grep -q "Artifact: manual_update" "$project/target/ops/runs/smoke-daily-ops/manual_update-explain.txt"
grep -q "mode       new_items_only" "$project/target/ops/runs/smoke-daily-ops/source-window.txt"
grep -q "ok source  questions files=3" "$project/target/ops/runs/smoke-daily-ops/source-window.txt"
grep -q '"eventType":"COMPLETE"' "$project/target/ops/runs/smoke-daily-ops/openlineage.ndjson"
grep -q '"resourceSpans"' "$project/target/ops/runs/smoke-daily-ops/otel.json"
grep -q "PASS    structural_artifacts" "$project/target/ops/runs/smoke-daily-ops/quality-gates.txt"
grep -q "PENDING domain_review" "$project/target/ops/runs/smoke-daily-ops/quality-gates.txt"
grep -q "state_and_artifacts_plus_run_bundle" "$project/target/ops/runs/smoke-daily-ops/archive.txt"
tar -tzf "$project/target/ops/archives/smoke-daily-ops/fbt-evidence.tar.gz" | grep -q ".fbt/state"
tar -tzf "$project/target/ops/archives/smoke-daily-ops/fbt-evidence.tar.gz" | grep -q ".fbt/artifacts"
tar -tzf "$project/target/ops/archives/smoke-daily-ops/fbt-evidence.tar.gz" | grep -q "target/ops/runs/smoke-daily-ops"
grep -q "fbt stops at artifacts" "$project/target/ops/runs/smoke-daily-ops/publish-handoff.txt"
grep -q "Review and publish outside fbt" "$project/target/ops/publish/smoke-daily-ops/notification.md"

echo "daily-ops-smoke: ok"
