#!/usr/bin/env sh
set -eu

project_dir=""
run_id=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --project-dir)
      shift
      project_dir="$1"
      ;;
    --run-id)
      shift
      run_id="$1"
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
  shift
done

if [ -z "$project_dir" ] || [ -z "$run_id" ]; then
  echo "usage: prepare-publish-handoff.sh --project-dir DIR --run-id ID" >&2
  exit 2
fi

run_dir="$project_dir/target/ops/runs/$run_id"
handoff_dir="$project_dir/target/ops/publish/$run_id"
mkdir -p "$handoff_dir"

quality_status="unknown"
if [ -f "$run_dir/quality-gates.json" ]; then
  quality_status="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["status"])' "$run_dir/quality-gates.json")"
fi

cat >"$handoff_dir/publish-manifest.json" <<EOF
{
  "schema_version": 1,
  "run_id": "$run_id",
  "quality_status": "$quality_status",
  "artifacts": [
    "target/artifacts/qa/latest/faq_candidates.md",
    "target/artifacts/qa/latest/manual_patch_candidates.md",
    "target/artifacts/qa/latest/unresolved_questions.md",
    "target/artifacts/manual/latest/manual_update.md"
  ],
  "evidence": [
    "target/ops/runs/$run_id/plan.txt",
    "target/ops/runs/$run_id/build.txt",
    "target/ops/runs/$run_id/manual_update-explain.txt",
    "target/ops/runs/$run_id/quality-gates.json",
    "target/ops/archives/$run_id/fbt-evidence.tar.gz"
  ],
  "fbt_boundary": "fbt produced artifacts and evidence; Git/PR/publisher workflows decide approval and publication"
}
EOF

cat >"$handoff_dir/pr-body.md" <<EOF
# Daily Support Knowledge Update

Run ID: \`$run_id\`
Quality status: \`$quality_status\`

## Generated Artifacts

- \`target/artifacts/qa/latest/faq_candidates.md\`
- \`target/artifacts/qa/latest/manual_patch_candidates.md\`
- \`target/artifacts/qa/latest/unresolved_questions.md\`
- \`target/artifacts/manual/latest/manual_update.md\`

## Evidence To Review

- \`target/ops/runs/$run_id/manual_update-explain.txt\`
- \`target/ops/runs/$run_id/quality-gates.txt\`
- \`target/ops/runs/$run_id/retention.txt\`
- \`target/ops/archives/$run_id/fbt-evidence.tar.gz\`

## Reviewer Checklist

- Confirm the manual update is grounded in the source questions and answers.
- Confirm unresolved questions are not published as final guidance.
- Confirm the archive bundle is attached to CI artifacts or external storage.

fbt does not approve or publish this change. Merge and publishing decisions
belong to the repository, CI, and knowledge-base workflow.
EOF

cat >"$handoff_dir/notification.md" <<EOF
Daily support knowledge artifacts are ready for review.

- run: $run_id
- quality: $quality_status
- manual update: target/artifacts/manual/latest/manual_update.md
- evidence: target/ops/runs/$run_id/manual_update-explain.txt
- archive: target/ops/archives/$run_id/fbt-evidence.tar.gz

Review and publish outside fbt.
EOF

printf 'Publish Handoff\n'
printf '  manifest      %s\n' "$handoff_dir/publish-manifest.json"
printf '  pr_body       %s\n' "$handoff_dir/pr-body.md"
printf '  notification  %s\n' "$handoff_dir/notification.md"
printf '  boundary      fbt stops at artifacts, evidence, and handoff files\n'
