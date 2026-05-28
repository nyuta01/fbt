#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FBT_BIN="${FBT_BIN:-$ROOT_DIR/bin/fbt}"
if [[ ! -x "$FBT_BIN" ]]; then
  echo "FBT_BIN is not executable: $FBT_BIN" >&2
  exit 1
fi

if [[ -z "${FBT_REAL_LLM_RUNNER_COMMAND:-}" ]]; then
  echo "real-llm-smoke: skipped (set FBT_REAL_LLM_RUNNER_COMMAND to opt in)"
  exit 0
fi

runner_name="${FBT_REAL_LLM_RUNNER_NAME:-real.llm}"
model_provider="${FBT_REAL_LLM_MODEL_PROVIDER:-external}"
model_name="${FBT_REAL_LLM_MODEL_NAME:-real-llm-smoke}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

project="$tmpdir/real_llm_smoke"
mkdir -p "$project"/{bin,sources,transforms,assets,policies,data/input}

export FBT_REAL_LLM_RUNNER_COMMAND
cat >"$project/bin/real-llm-runner" <<'SH'
#!/usr/bin/env sh
exec sh -c "$FBT_REAL_LLM_RUNNER_COMMAND"
SH
chmod +x "$project/bin/real-llm-runner"

cat >"$project/fs_project.yml" <<YAML
name: real_llm_smoke
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
artifact_path: "target/artifacts"
runners:
  - name: $runner_name
    type: llm
    protocol: stdio_jsonrpc
    command: bin/real-llm-runner
YAML

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
  - name: real_llm_smoke_scope
    read: ["data/input/"]
    write: [".fbt/work/", "target/artifacts/"]
    network: true
YAML

cat >"$project/transforms/smoke.yml" <<YAML
transforms:
  - name: real_llm_smoke
    type: llm
    runner: $runner_name
    model:
      provider: $model_provider
      name: $model_name
    inputs:
      - source: input.notes
    outputs:
      - name: result
        type: markdown_directory
        path: target/artifacts/result/
    assets:
      - ref: smoke_prompt
    policy: real_llm_smoke_scope
YAML

printf '{"id":"N-1","summary":"real LLM smoke input"}\n' >"$project/data/input/notes.jsonl"

"$FBT_BIN" doctor --project-dir "$project" >"$tmpdir/doctor.txt"
grep -q "Doctor: ok" "$tmpdir/doctor.txt"
grep -q "RUNNER_PROTOCOL_OK" "$tmpdir/doctor.txt"

"$FBT_BIN" build --project-dir "$project" --select real_llm_smoke >"$tmpdir/build.txt"
grep -q "committed:" "$tmpdir/build.txt"
test -d "$project/target/artifacts/result"

"$FBT_BIN" artifact show result --project-dir "$project" >"$tmpdir/artifact-show.txt"
grep -q "artifact.real_llm_smoke.result" "$tmpdir/artifact-show.txt"

echo "real-llm-smoke: ok"
