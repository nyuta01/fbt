#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

fbt_bin="$tmpdir/fbt"
go build -o "$fbt_bin" ./cmd/fbt

project="$tmpdir/knowledge_ops"
cd "$tmpdir"

"$fbt_bin" init "$project" --template support >"$tmpdir/init.txt"
grep -q "Initialized support project" "$tmpdir/init.txt"

"$fbt_bin" parse --project-dir "$project" >"$tmpdir/parse.txt"
grep -q "Manifest written" "$tmpdir/parse.txt"

"$fbt_bin" doctor --project-dir "$project" >"$tmpdir/doctor.txt"
grep -q "Doctor: ok" "$tmpdir/doctor.txt"

"$fbt_bin" plan --project-dir "$project" --select case_summaries >"$tmpdir/plan-case.txt"
grep -q "run transform.knowledge_ops.case_summaries" "$tmpdir/plan-case.txt"

"$fbt_bin" build --project-dir "$project" --select case_summaries >"$tmpdir/build-case.txt"
grep -q "committed:" "$tmpdir/build-case.txt"
test -f "$project/target/artifacts/support/case_summaries/index.md"

"$fbt_bin" artifact path case_summaries --project-dir "$project" >"$tmpdir/artifact-path.txt"
grep -q "logical_path: target/artifacts/support/case_summaries" "$tmpdir/artifact-path.txt"
grep -q "storage_path: .fbt/artifacts/" "$tmpdir/artifact-path.txt"
"$fbt_bin" artifact show case_summaries --project-dir "$project" >"$tmpdir/artifact-show.txt"
grep -q "generated_by: transform_run.run_" "$tmpdir/artifact-show.txt"
grep -q "semantic_descriptor:" "$tmpdir/artifact-show.txt"
"$fbt_bin" artifact history case_summaries --project-dir "$project" >"$tmpdir/artifact-history.txt"
grep -q "artifact_version.knowledge_ops.case_summaries" "$tmpdir/artifact-history.txt"

"$fbt_bin" export openlineage --project-dir "$project" --output "$tmpdir/openlineage.ndjson" >"$tmpdir/export-openlineage.txt"
grep -q "OpenLineage events written" "$tmpdir/export-openlineage.txt"
grep -q '"eventType":"COMPLETE"' "$tmpdir/openlineage.ndjson"
grep -q '"name":"transform.knowledge_ops.case_summaries"' "$tmpdir/openlineage.ndjson"
grep -q '"fbt_artifact"' "$tmpdir/openlineage.ndjson"

"$fbt_bin" review status case_summaries --project-dir "$project" >"$tmpdir/review-status.txt"
grep -q "status: pending" "$tmpdir/review-status.txt"
grep -q "next: fbt review show case_summaries" "$tmpdir/review-status.txt"
"$fbt_bin" review show case_summaries --project-dir "$project" >"$tmpdir/review-show.txt"
grep -q "inspect: fbt artifact show case_summaries" "$tmpdir/review-show.txt"
grep -q "approve_after_review: fbt review approve case_summaries" "$tmpdir/review-show.txt"

"$fbt_bin" review approve case_summaries --project-dir "$project" --comment "smoke" >"$tmpdir/review-approve.txt"
grep -q "status: approved" "$tmpdir/review-approve.txt"

"$fbt_bin" build --project-dir "$project" --select weekly_support_insights >"$tmpdir/build-weekly.txt"
grep -q "committed:" "$tmpdir/build-weekly.txt"
test -f "$project/target/artifacts/support/weekly_insights.md"

"$fbt_bin" docs generate --project-dir "$project" >"$tmpdir/docs.txt"
grep -q "Docs written" "$tmpdir/docs.txt"
test -f "$project/target/docs/index.md"

echo "knowledge-loop-smoke: ok"
