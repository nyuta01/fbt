#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
default_project_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)

project_dir="${FBT_PROJECT_DIR:-${1:-$default_project_dir}}"
selector="${FBT_SELECTOR:-tag:daily_qa}"
ready_file="${FBT_READY_FILE:-$project_dir/data/qa/inbox/_READY}"
fbt_bin="${FBT_BIN:-fbt}"
run_id="${FBT_RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
ops_root="$project_dir/target/ops"
run_dir="$ops_root/runs/$run_id"
latest_dir="$ops_root/latest"

run_fbt() {
  # FBT_BIN may intentionally contain arguments, for example: go run ./cmd/fbt.
  # shellcheck disable=SC2086
  $fbt_bin "$@" --project-dir "$project_dir"
}

if [ ! -f "$ready_file" ]; then
  printf '%s\n' "source window is not ready: $ready_file" >&2
  printf '%s\n' "prepare the daily source files before running fbt" >&2
  exit 4
fi

mkdir -p "$run_dir"

"$script_dir/check-source-window.py" --project-dir "$project_dir" --ready-file "$ready_file" >"$run_dir/source-window.txt"
run_fbt doctor >"$run_dir/doctor.txt"
run_fbt plan --select "$selector" >"$run_dir/plan.txt"
run_fbt build --select "$selector" >"$run_dir/build.txt"
run_fbt artifact show faq_candidates >"$run_dir/faq_candidates-show.txt"
run_fbt artifact show manual_update >"$run_dir/manual_update-show.txt"
run_fbt artifact explain manual_update >"$run_dir/manual_update-explain.txt"
run_fbt artifact retention >"$run_dir/retention.txt"
run_fbt export openlineage --output "$run_dir/openlineage.ndjson" >"$run_dir/openlineage-export.txt"
run_fbt export otel --output "$run_dir/otel.json" >"$run_dir/otel-export.txt"
"$script_dir/check-quality-gates.py" --project-dir "$project_dir" --run-dir "$run_dir" >"$run_dir/quality-gates.txt"
"$script_dir/prepare-publish-handoff.sh" --project-dir "$project_dir" --run-id "$run_id" >"$run_dir/publish-handoff.txt"
"$script_dir/check-security-profile.py" --project-dir "$project_dir" --run-id "$run_id" >"$run_dir/security-profile.txt"
"$script_dir/archive-fbt-evidence.sh" --project-dir "$project_dir" --run-id "$run_id" >"$run_dir/archive.txt"

cat >"$run_dir/summary.md" <<EOF
# fbt Daily Support Knowledge Run

- run id: $run_id
- project: $project_dir
- selector: $selector
- readiness marker: $ready_file

## Produced By fbt

- target/artifacts/qa/latest/faq_candidates.md
- target/artifacts/qa/latest/manual_patch_candidates.md
- target/artifacts/qa/latest/unresolved_questions.md
- target/artifacts/manual/latest/manual_update.md

## Operational Evidence

- source-window.txt: ingestion-owned window manifest validation
- doctor.txt: local readiness and runner diagnostics
- plan.txt: run, skip, and block decisions before runner execution
- build.txt: committed artifact versions and next inspection commands
- manual_update-explain.txt: source, asset, runner, policy, and eval lineage
- retention.txt: local state and artifact archive boundary
- openlineage.ndjson: standard lineage events
- otel.json: OTLP/JSON trace payload
- quality-gates.txt/json: structural and evidence gates plus review handoff
- publish-handoff.txt: PR, publish, and notification draft locations
- security-profile.txt: external sandbox profile and secret handoff scan
- archive.txt: CI/storage handoff for state, artifacts, and this run bundle

Approval, publishing, notifications, and scheduling stay outside fbt. In a
production job, hand this run directory to Git, CI artifacts, Slack, or your
knowledge-base publishing workflow.
EOF

rm -rf "$latest_dir"
mkdir -p "$latest_dir"
for file in "$run_dir"/*; do
  cp "$file" "$latest_dir/"
done

printf 'fbt daily support knowledge run complete\n'
printf 'run directory: %s\n' "$run_dir"
printf 'latest bundle: %s\n' "$latest_dir"
