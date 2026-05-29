#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

go run ./cmd/fbt --help >"$tmpdir/help.txt"
grep -q "versioned filesystem artifacts" "$tmpdir/help.txt"
if grep -q "Planned commands" "$tmpdir/help.txt"; then
  echo "help should not list placeholder commands" >&2
  exit 1
fi

go run ./cmd/fbt version >"$tmpdir/version.txt"
grep -q "^fbt 0.2.1$" "$tmpdir/version.txt"

project="$tmpdir/project"
mkdir -p "$project"/{bin,sources,transforms,assets,policies,evals,prompts,data/support/tickets}
cat >"$project/bin/fbt-fake-runner" <<SH
#!/usr/bin/env sh
exec go run "$ROOT_DIR/tests/runner_fixtures/fake"
SH
chmod +x "$project/bin/fbt-fake-runner"
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
    command: bin/fbt-fake-runner
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
      sections: ["Fake Output"]
    grants_confidence: structural
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

go run ./cmd/fbt doctor --project-dir "$project" >"$tmpdir/doctor.txt"
grep -q "Doctor: ok" "$tmpdir/doctor.txt"
grep -q "RUNNER_PROTOCOL_OK" "$tmpdir/doctor.txt"

go run ./cmd/fbt plan --project-dir "$project" --select tag:support >"$tmpdir/plan.txt"
grep -q "selected  1" "$tmpdir/plan.txt"
grep -q "RUN     case_summaries" "$tmpdir/plan.txt"
test ! -f "$project/.fbt/state/manifest.json"

go run ./cmd/fbt artifact ls --project-dir "$project" >"$tmpdir/artifact-ls.txt"

go run ./cmd/fbt build --project-dir "$project" >"$tmpdir/build.txt"
grep -q "committed  " "$tmpdir/build.txt"
test -f "$project/.fbt/state/manifest.json"
test -f "$project/target/artifacts/support/case_summaries/index.md"
go run ./cmd/fbt artifact show case_summaries --project-dir "$project" >"$tmpdir/artifact-show.txt"
grep -q "Semantic summary  " "$tmpdir/artifact-show.txt"

go run ./cmd/fbt export openlineage --project-dir "$project" --output "$tmpdir/openlineage.ndjson" >"$tmpdir/export-openlineage.txt"
grep -q "Export: openlineage" "$tmpdir/export-openlineage.txt"
grep -q "OpenLineage RunEvent NDJSON" "$tmpdir/export-openlineage.txt"
grep -q '"eventType":"COMPLETE"' "$tmpdir/openlineage.ndjson"
grep -q '"fbt_evaluations"' "$tmpdir/openlineage.ndjson"

go run ./cmd/fbt export otel --project-dir "$project" --output "$tmpdir/otel.json" >"$tmpdir/export-otel.txt"
grep -q "Export: otel" "$tmpdir/export-otel.txt"
grep -q "OpenTelemetry OTLP/JSON traces" "$tmpdir/export-otel.txt"
grep -q '"resourceSpans"' "$tmpdir/otel.json"
grep -q '"fbt.transform.id"' "$tmpdir/otel.json"
grep -q '"progress"' "$tmpdir/otel.json"

for command in parse eval docs state runner review; do
  if go run ./cmd/fbt "$command" --project-dir "$project" >"$tmpdir/$command.out" 2>"$tmpdir/$command.err"; then
    echo "$command should not be part of the fbt command surface" >&2
    exit 1
  fi
  grep -q "unknown command \"$command\"" "$tmpdir/$command.err"
done

if go run ./cmd/fbt run >"$tmpdir/run.out" 2>"$tmpdir/run.err"; then
  echo "expected fbt run to be outside the MVP command surface" >&2
  exit 1
fi
grep -q "unknown command \"run\"" "$tmpdir/run.err"

echo "cli-smoke: ok"
