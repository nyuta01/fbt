#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

project="$tmpdir/retention_volume"
mkdir -p "$project"/{bin,sources,transforms,policies,data/input}

cat >"$project/bin/fbt-command-runner" <<'SH'
#!/usr/bin/env sh
set -eu

if [ -n "${FBT_SOURCE_ROOT:-}" ]; then
  runner_dir="$FBT_SOURCE_ROOT"
else
  script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
  runner_dir="$script_dir/../../.."
fi

export FBT_COMMAND_WORKDIR="${FBT_COMMAND_WORKDIR:-$PWD}"
cd "$runner_dir"
exec go run ./adapters/command/cmd/fbt-runner-command
SH
chmod +x "$project/bin/fbt-command-runner"

cat >"$project/bin/render-summary" <<'SH'
#!/usr/bin/env sh
set -eu

out="$FBT_WORK_OUTPUTS/retention_summary"
{
  printf '# Retention Volume Summary\n\n'
  printf '## Source Files\n\n'
  find data/input -type f -name '*.md' | sort | while IFS= read -r file; do
    printf -- '- `%s`: ' "$file"
    head -n 1 "$file"
  done
} >"$out"
SH
chmod +x "$project/bin/render-summary"

cat >"$project/fs_project.yml" <<'YAML'
name: retention_volume
config_version: 1
version: 0.1.0
source_paths: ["sources"]
transform_paths: ["transforms"]
policy_paths: ["policies"]
artifact_path: "target/artifacts"
state:
  backend: local
  path: .fbt/state
runners:
  - name: local.command
    type: command
    protocol: stdio_jsonrpc
    command: bin/fbt-command-runner
    cwd: .
    env:
      - FBT_SOURCE_ROOT
selectors:
  - name: retention_volume
    definition:
      method: tag
      value: retention_volume
YAML

cat >"$project/sources/input.yml" <<'YAML'
sources:
  - name: input
    artifacts:
      - name: notes
        type: markdown_directory
        path: data/input/
        tags: ["retention_volume"]
YAML

cat >"$project/policies/summary.yml" <<'YAML'
policies:
  - name: summary_scope
    read: ["data/input/"]
    write: [".fbt/work/", "target/artifacts/"]
    network: false
YAML

cat >"$project/transforms/summary.yml" <<'YAML'
transforms:
  - name: retention_summary
    type: command
    runner: local.command
    command: ["bin/render-summary"]
    inputs:
      - source: input.notes
    outputs:
      - name: retention_summary
        type: markdown
        path: target/artifacts/retention/summary.md
    policy: summary_scope
    tags: ["retention_volume"]
YAML

for day in 1 2 3 4 5 6 7 8; do
  printf '# Day %02d source\n\nRetention fixture source batch %02d.\n' "$day" "$day" >"$project/data/input/day-$day.md"
  FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt build --project-dir "$project" --select retention_summary >"$tmpdir/build-$day.txt"
  grep -q "SUCCESS retention_summary" "$tmpdir/build-$day.txt"
done

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt artifact retention --project-dir "$project" >"$tmpdir/retention.txt"
grep -Eq "Policy +keep_all" "$tmpdir/retention.txt"
grep -Eq "Archive unit +\\.fbt/state \\+ \\.fbt/artifacts" "$tmpdir/retention.txt"
grep -Eq "Artifact versions +8" "$tmpdir/retention.txt"
grep -Eq "Current versions +1" "$tmpdir/retention.txt"
grep -Eq "Historical versions +7" "$tmpdir/retention.txt"
grep -Eq "Protected versions +1 current pointer\\(s\\)" "$tmpdir/retention.txt"
grep -Eq "Prune +not supported in MVP; future prune must dry-run first" "$tmpdir/retention.txt"
grep -Eq "Action +no files removed; archive state and artifact dirs together" "$tmpdir/retention.txt"

FBT_SOURCE_ROOT="$ROOT_DIR" go run ./cmd/fbt artifact retention --project-dir "$project" --json >"$tmpdir/retention.json"
python3 - "$tmpdir/retention.json" <<'PY'
import json
import sys

payload = json.load(open(sys.argv[1], encoding="utf-8"))
report = payload["retention"]
assert report["policy"] == "keep_all"
assert report["archive_unit"] == "state_and_artifacts"
assert report["artifact_versions"] == 8
assert report["current_versions"] == 1
assert report["historical_versions"] == 7
assert len(report["protected_version_ids"]) == 1
assert report["prune_supported"] is False
assert report["dry_run_required"] is True
roots = set(report["archive_roots"])
assert any(root.endswith(".fbt/state") for root in roots)
assert any(root.endswith(".fbt/artifacts") for root in roots)
PY

echo "retention-high-volume-smoke: ok"
