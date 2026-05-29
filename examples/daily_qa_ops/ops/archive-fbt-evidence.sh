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
  echo "usage: archive-fbt-evidence.sh --project-dir DIR --run-id ID" >&2
  exit 2
fi

run_dir="$project_dir/target/ops/runs/$run_id"
archive_dir="$project_dir/target/ops/archives/$run_id"
archive_path="$archive_dir/fbt-evidence.tar.gz"
manifest_path="$archive_dir/archive-manifest.json"

test -d "$project_dir/.fbt/state"
test -d "$project_dir/.fbt/artifacts"
test -d "$run_dir"

mkdir -p "$archive_dir"

cat >"$manifest_path" <<EOF
{
  "schema_version": 1,
  "archive_unit": "state_and_artifacts_plus_run_bundle",
  "run_id": "$run_id",
  "roots": [
    ".fbt/state",
    ".fbt/artifacts",
    "target/ops/runs/$run_id"
  ],
  "restore_note": "Restore these roots together into the same project checkout before running fbt artifact explain, diff, export openlineage, or export otel.",
  "prune_supported": false,
  "dry_run_required": true
}
EOF

tar -czf "$archive_path" \
  -C "$project_dir" \
  .fbt/state \
  .fbt/artifacts \
  "target/ops/runs/$run_id"

printf 'Archive\n'
printf '  archive_unit  state_and_artifacts_plus_run_bundle\n'
printf '  archive       %s\n' "$archive_path"
printf '  manifest      %s\n' "$manifest_path"
printf '  restore       restore .fbt/state, .fbt/artifacts, and target/ops/runs/%s together\n' "$run_id"
printf '  prune         unsupported; archive first and keep current pointers protected\n'
