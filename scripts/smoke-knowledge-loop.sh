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

"$fbt_bin" doctor --project-dir "$project" >"$tmpdir/doctor.txt"
grep -q "Doctor: ok" "$tmpdir/doctor.txt"

"$fbt_bin" plan --project-dir "$project" --select case_summaries >"$tmpdir/plan-case.txt"
grep -q "run transform.knowledge_ops.case_summaries" "$tmpdir/plan-case.txt"

"$fbt_bin" plan --project-dir "$project" --select +weekly_support_insights >"$tmpdir/plan-upstream.txt"
grep -q "Plan: 2 selected, 1 run, 0 skipped, 1 blocked" "$tmpdir/plan-upstream.txt"
grep -q "run transform.knowledge_ops.case_summaries" "$tmpdir/plan-upstream.txt"
grep -q "blocked transform.knowledge_ops.weekly_support_insights" "$tmpdir/plan-upstream.txt"

"$fbt_bin" build --project-dir "$project" --select case_summaries >"$tmpdir/build-case.txt"
grep -q "committed:" "$tmpdir/build-case.txt"
test -f "$project/target/artifacts/support/case_summaries/index.md"

"$fbt_bin" plan --project-dir "$project" --select case_summaries+ >"$tmpdir/plan-downstream.txt"
grep -q "Plan: 2 selected, 1 run, 1 skipped, 0 blocked" "$tmpdir/plan-downstream.txt"
grep -q "skip transform.knowledge_ops.case_summaries" "$tmpdir/plan-downstream.txt"
grep -q "run transform.knowledge_ops.weekly_support_insights" "$tmpdir/plan-downstream.txt"

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

"$fbt_bin" artifact explain case_summaries --project-dir "$project" >"$tmpdir/artifact-explain.txt"
grep -q "current_version: artifact_version.knowledge_ops.case_summaries" "$tmpdir/artifact-explain.txt"

"$fbt_bin" build --project-dir "$project" --select weekly_support_insights >"$tmpdir/build-weekly.txt"
grep -q "committed:" "$tmpdir/build-weekly.txt"
test -f "$project/target/artifacts/support/weekly_insights.md"

echo "knowledge-loop-smoke: ok"
