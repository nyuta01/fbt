#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

go run ./cmd/fbt --help >"$tmpdir/help.txt"
grep -q "file build tool" "$tmpdir/help.txt"
grep -q "Planned commands" "$tmpdir/help.txt"

go run ./cmd/fbt version >"$tmpdir/version.txt"
grep -q "^fbt 0.0.0-dev$" "$tmpdir/version.txt"

project="$tmpdir/project"
mkdir -p "$project"/{sources,transforms,assets,policies,evals,prompts,data/support/tickets}
cat >"$project/fs_project.yml" <<'YAML'
name: knowledge_ops
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
artifact_path: "target/artifacts"
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
YAML
cat >"$project/sources/support.yml" <<'YAML'
sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
        tags: ["support"]
YAML
cat >"$project/assets/assets.yml" <<'YAML'
assets:
  - name: case_summary_prompt
    type: prompt
    path: prompts/case_summary.md
YAML
cat >"$project/policies/support.yml" <<'YAML'
policies:
  - name: support_agent_scope
    read: ["data/support/"]
    write: [".fbt/work/", "target/artifacts/support/"]
    network: true
YAML
cat >"$project/evals/support.yml" <<'YAML'
evals:
  - name: required_case_sections
    type: deterministic
    config:
      sections: ["Summary"]
YAML
cat >"$project/transforms/case.yml" <<'YAML'
transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    assets:
      - ref: case_summary_prompt
    policy: support_agent_scope
    evals:
      - required_case_sections
    tags: ["support"]
YAML
printf '{}\n' >"$project/data/support/tickets/2026-05-28.jsonl"
printf '# Task\n' >"$project/prompts/case_summary.md"

go run ./cmd/fbt parse --project-dir "$project" >"$tmpdir/parse.txt"
grep -q "Manifest written" "$tmpdir/parse.txt"
test -f "$project/.fbt/state/manifest.json"

go run ./cmd/fbt plan --project-dir "$project" --select tag:support >"$tmpdir/plan.txt"
grep -q "Plan: 1 selected" "$tmpdir/plan.txt"
grep -q "run transform.knowledge_ops.case_summaries" "$tmpdir/plan.txt"

go run ./cmd/fbt state status --project-dir "$project" >"$tmpdir/state.txt"
grep -q "State dir:" "$tmpdir/state.txt"

go run ./cmd/fbt artifact ls --project-dir "$project" >"$tmpdir/artifact-ls.txt"

go run ./cmd/fbt runner list --project-dir "$project" >"$tmpdir/runner-list.txt"
grep -q "openai.responses" "$tmpdir/runner-list.txt"

if go run ./cmd/fbt build >"$tmpdir/build.out" 2>"$tmpdir/build.err"; then
  echo "expected fbt build to be a planned-but-unimplemented command" >&2
  exit 1
fi
grep -q "not implemented yet" "$tmpdir/build.err"

echo "cli-smoke: ok"
