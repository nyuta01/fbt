#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FBT_BIN="${FBT_BIN:-$ROOT_DIR/bin/fbt}"
if [[ ! -x "$FBT_BIN" ]]; then
  echo "FBT_BIN is not executable: $FBT_BIN" >&2
  exit 1
fi

matrix="${FBT_RUNNER_ADAPTER_SMOKE_MATRIX:-}"
if [[ -z "$matrix" ]]; then
  cat <<'TEXT'
runner-adapter-smoke: skipped
set FBT_RUNNER_ADAPTER_SMOKE_MATRIX to opt in.

row format:
  logical_name|runner_type|artifact_type|command|required_env_csv|agent_adapter

example:
  openai.responses|llm|markdown|fbt-runner-openai responses|OPENAI_API_KEY|false
  codex.cli|agent|markdown|fbt-runner-codex-cli --profile fbt|OPENAI_API_KEY|true
TEXT
  exit 0
fi

timeout_seconds="${FBT_RUNNER_ADAPTER_SMOKE_TIMEOUT_SECONDS:-60}"
run_build="${FBT_RUNNER_ADAPTER_SMOKE_BUILD:-0}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

is_directory_artifact() {
  local artifact_type="$1"
  [[ "$artifact_type" == "directory" || "$artifact_type" == *_directory ]]
}

require_env_vars() {
  local row_name="$1"
  local env_csv="$2"
  local env_name
  IFS=',' read -r -a env_names <<<"$env_csv"
  for env_name in "${env_names[@]}"; do
    env_name="$(trim "$env_name")"
    [[ -z "$env_name" ]] && continue
    if [[ ! "$env_name" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
      echo "runner-adapter-smoke: $row_name has invalid env name: $env_name" >&2
      exit 2
    fi
    if [[ -z "${!env_name:-}" ]]; then
      echo "runner-adapter-smoke: $row_name missing required env: $env_name" >&2
      exit 2
    fi
  done
}

write_project() {
  local project="$1"
  local logical_name="$2"
  local runner_type="$3"
  local artifact_type="$4"
  local env_csv="$5"
  local output_path
  local env_name

  mkdir -p "$project"/{bin,sources,transforms,assets,policies,data/input}

  cat >"$project/bin/fbt-adapter-smoke-runner" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
exec sh -c "$FBT_RUNNER_ADAPTER_SMOKE_COMMAND"
SH
  chmod +x "$project/bin/fbt-adapter-smoke-runner"

  cat >"$project/fs_project.yml" <<YAML
name: runner_adapter_smoke
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
artifact_path: "target/artifacts"
runners:
  - name: $logical_name
    type: $runner_type
    protocol: stdio_jsonrpc
    command: bin/fbt-adapter-smoke-runner
    env:
      - FBT_RUNNER_ADAPTER_SMOKE_COMMAND
YAML

  IFS=',' read -r -a env_names <<<"$env_csv"
  for env_name in "${env_names[@]}"; do
    env_name="$(trim "$env_name")"
    [[ -z "$env_name" ]] && continue
    echo "      - $env_name" >>"$project/fs_project.yml"
  done

  cat >"$project/sources/input.yml" <<'YAML'
sources:
  - name: input
    artifacts:
      - name: notes
        type: jsonl_directory
        path: data/input/*.jsonl
YAML

  cat >"$project/assets/prompt.yml" <<'YAML'
assets:
  - name: smoke_prompt
    type: prompt
    path: assets/smoke_prompt.md
YAML

  cat >"$project/assets/smoke_prompt.md" <<'MD'
# Task

Write a short Markdown artifact with a `# Result` heading. Do not include
credentials, environment variable values, or raw tool outputs.
MD

  cat >"$project/policies/smoke.yml" <<'YAML'
policies:
  - name: adapter_smoke_scope
    read: ["data/input/", "assets/"]
    write: [".fbt/work/", "target/artifacts/"]
    network: true
    tools:
      allow: ["read_artifact", "search_project"]
      deny: ["write_source_files"]
    limits:
      timeout_seconds: 120
      max_output_bytes: 1048576
YAML

  if is_directory_artifact "$artifact_type"; then
    output_path="target/artifacts/result/"
  else
    output_path="target/artifacts/result.md"
  fi

  cat >"$project/transforms/smoke.yml" <<YAML
transforms:
  - name: adapter_smoke
    type: $runner_type
    runner: $logical_name
    model:
      provider: adapter-smoke
      name: $logical_name
    inputs:
      - source: input.notes
    outputs:
      - name: result
        type: $artifact_type
        path: $output_path
    assets:
      - ref: smoke_prompt
    policy: adapter_smoke_scope
YAML

  printf '{"id":"N-1","summary":"adapter smoke input"}\n' >"$project/data/input/notes.jsonl"
}

run_row() {
  local row_number="$1"
  local row="$2"
  local logical_name runner_type artifact_type command env_csv agent_adapter
  IFS='|' read -r logical_name runner_type artifact_type command env_csv agent_adapter <<<"$row"

  logical_name="$(trim "${logical_name:-}")"
  runner_type="$(trim "${runner_type:-}")"
  artifact_type="$(trim "${artifact_type:-}")"
  command="$(trim "${command:-}")"
  env_csv="$(trim "${env_csv:-}")"
  agent_adapter="$(trim "${agent_adapter:-false}")"

  if [[ -z "$logical_name" || -z "$runner_type" || -z "$artifact_type" || -z "$command" ]]; then
    echo "runner-adapter-smoke: row $row_number is invalid: $row" >&2
    exit 2
  fi
  if [[ "$agent_adapter" != "true" && "$agent_adapter" != "false" ]]; then
    echo "runner-adapter-smoke: $logical_name agent_adapter must be true or false" >&2
    exit 2
  fi

  require_env_vars "$logical_name" "$env_csv"

  local conformance_args=(
    tests/runner-conformance/run.py
    --runner-command "$command"
    --transform-type "$runner_type"
    --artifact-type "$artifact_type"
    --timeout-seconds "$timeout_seconds"
    --strict
  )
  if [[ "$agent_adapter" == "true" ]]; then
    conformance_args+=(--agent-adapter)
  fi

  echo "runner-adapter-smoke: conformance $logical_name"
  python3 "${conformance_args[@]}"

  local project="$tmpdir/project_$row_number"
  write_project "$project" "$logical_name" "$runner_type" "$artifact_type" "$env_csv"

  echo "runner-adapter-smoke: doctor $logical_name"
  FBT_RUNNER_ADAPTER_SMOKE_COMMAND="$command" "$FBT_BIN" doctor --project-dir "$project" >"$tmpdir/doctor_$row_number.txt"
  grep -q "Doctor: ok" "$tmpdir/doctor_$row_number.txt"
  grep -q "RUNNER_PROTOCOL_OK" "$tmpdir/doctor_$row_number.txt"

  echo "runner-adapter-smoke: plan $logical_name"
  FBT_RUNNER_ADAPTER_SMOKE_COMMAND="$command" "$FBT_BIN" plan --project-dir "$project" --select adapter_smoke >"$tmpdir/plan_$row_number.txt"
  grep -q "RUN     adapter_smoke" "$tmpdir/plan_$row_number.txt"

  if [[ "$run_build" == "1" ]]; then
    echo "runner-adapter-smoke: build $logical_name"
    FBT_RUNNER_ADAPTER_SMOKE_COMMAND="$command" "$FBT_BIN" build --project-dir "$project" --select adapter_smoke >"$tmpdir/build_$row_number.txt"
    grep -q "committed  " "$tmpdir/build_$row_number.txt"
    FBT_RUNNER_ADAPTER_SMOKE_COMMAND="$command" "$FBT_BIN" artifact show result --project-dir "$project" >"$tmpdir/artifact_$row_number.txt"
    grep -q "artifact.runner_adapter_smoke.result" "$tmpdir/artifact_$row_number.txt"
  fi
}

row_number=0
while IFS= read -r raw_row || [[ -n "$raw_row" ]]; do
  row="$(trim "$raw_row")"
  [[ -z "$row" || "$row" == \#* ]] && continue
  row_number=$((row_number + 1))
  run_row "$row_number" "$row"
done <<<"$matrix"

if [[ "$row_number" -eq 0 ]]; then
  echo "runner-adapter-smoke: no matrix rows after filtering" >&2
  exit 2
fi

echo "runner-adapter-smoke: ok ($row_number rows)"
