#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

FBT_BIN="${FBT_BIN:-$ROOT_DIR/bin/fbt}"
if [[ ! -x "$FBT_BIN" ]]; then
  echo "FBT_BIN is not executable: $FBT_BIN" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

schema_missing="$tmpdir/schema-missing"
mkdir -p "$schema_missing"
cat >"$schema_missing/fs_project.yml" <<'YAML'
name: schema_missing
YAML
set +e
"$FBT_BIN" parse --project-dir "$schema_missing" >"$tmpdir/schema-missing.out" 2>"$tmpdir/schema-missing.err"
schema_missing_code=$?
set -e
if [[ "$schema_missing_code" -ne 2 ]]; then
  echo "expected missing config_version parse exit code 2, got $schema_missing_code" >&2
  exit 1
fi
grep -q "CONFIG_VERSION_MISSING" "$tmpdir/schema-missing.err"

schema_unsupported="$tmpdir/schema-unsupported"
mkdir -p "$schema_unsupported"
cat >"$schema_unsupported/fs_project.yml" <<'YAML'
name: schema_unsupported
config_version: 999
YAML
set +e
"$FBT_BIN" parse --project-dir "$schema_unsupported" >"$tmpdir/schema-unsupported.out" 2>"$tmpdir/schema-unsupported.err"
schema_unsupported_code=$?
set -e
if [[ "$schema_unsupported_code" -ne 2 ]]; then
  echo "expected unsupported config_version parse exit code 2, got $schema_unsupported_code" >&2
  exit 1
fi
grep -q "CONFIG_VERSION_UNSUPPORTED" "$tmpdir/schema-unsupported.err"

happy="$tmpdir/happy"
"$FBT_BIN" init "$happy" --template support >"$tmpdir/init-happy.txt"
redaction_marker="FBT_CONFORMANCE_SECRET_DO_NOT_EXPORT"
printf '\n- Do not export marker: %s\n' "$redaction_marker" >>"$happy/assets/support_style_guide.md"
printf '{"id":"T-secret","summary":"%s","impact":"redaction fixture"}\n' "$redaction_marker" >>"$happy/data/support/tickets/2026-05-28.jsonl"
"$FBT_BIN" build --project-dir "$happy" --select case_summaries >"$tmpdir/build-case.txt"
test -f "$happy/target/artifacts/support/case_summaries/index.md"
"$FBT_BIN" artifact show case_summaries --project-dir "$happy" >"$tmpdir/artifact-show.txt"
grep -q "semantic_descriptor:" "$tmpdir/artifact-show.txt"
test -f "$happy/.fbt/state/policy_decisions.json"
grep -q '"status": "allowed"' "$happy/.fbt/state/policy_decisions.json"

"$FBT_BIN" build --project-dir "$happy" --select case_summaries >"$tmpdir/build-case-again.txt"
grep -q "Build: 1 selected, 0 run, 1 skipped, 0 blocked" "$tmpdir/build-case-again.txt"

set +e
"$FBT_BIN" build --project-dir "$happy" --select weekly_support_insights >"$tmpdir/build-weekly-blocked.txt" 2>"$tmpdir/build-weekly-blocked.err"
blocked_code=$?
set -e
if [[ "$blocked_code" -ne 3 ]]; then
  echo "expected downstream build to be blocked before approval, got $blocked_code" >&2
  cat "$tmpdir/build-weekly-blocked.txt" >&2
  cat "$tmpdir/build-weekly-blocked.err" >&2
  exit 1
fi
grep -q "next: fbt review approve case_summaries" "$tmpdir/build-weekly-blocked.txt"

capability_mismatch="$tmpdir/capability-mismatch"
"$FBT_BIN" init "$capability_mismatch" --template support >"$tmpdir/init-capability-mismatch.txt"
cat >"$capability_mismatch/bin/fbt-demo-llm-runner" <<EOF_RUNNER
#!/usr/bin/env sh
export FBT_FAKE_RUNNER_ARTIFACT_TYPES=text
exec go run "$ROOT_DIR/runners/fake" "\$@"
EOF_RUNNER
chmod +x "$capability_mismatch/bin/fbt-demo-llm-runner"
set +e
"$FBT_BIN" build --project-dir "$capability_mismatch" --select case_summaries >"$tmpdir/build-capability-mismatch.txt" 2>"$tmpdir/build-capability-mismatch.err"
capability_mismatch_code=$?
set -e
if [[ "$capability_mismatch_code" -ne 6 ]]; then
  echo "expected runner capability mismatch exit code 6, got $capability_mismatch_code" >&2
  cat "$tmpdir/build-capability-mismatch.txt" >&2
  cat "$tmpdir/build-capability-mismatch.err" >&2
  exit 1
fi
grep -q "runner capability incompatible" "$tmpdir/build-capability-mismatch.err"

candidate_escape="$tmpdir/candidate-escape"
"$FBT_BIN" init "$candidate_escape" --template support >"$tmpdir/init-candidate-escape.txt"
cat >"$candidate_escape/bin/fbt-demo-llm-runner" <<EOF_RUNNER
#!/usr/bin/env sh
export FBT_FAKE_RUNNER_OUTPUT_OUTSIDE_WORK=1
exec go run "$ROOT_DIR/runners/fake" "\$@"
EOF_RUNNER
chmod +x "$candidate_escape/bin/fbt-demo-llm-runner"
set +e
"$FBT_BIN" build --project-dir "$candidate_escape" --select case_summaries >"$tmpdir/build-candidate-escape.txt" 2>"$tmpdir/build-candidate-escape.err"
candidate_escape_code=$?
set -e
if [[ "$candidate_escape_code" -eq 0 ]]; then
  echo "expected output candidate outside work dir to fail" >&2
  exit 1
fi
grep -q "output candidate outside work outputs" "$tmpdir/build-candidate-escape.err"
if [[ -e "$candidate_escape/target/artifacts/support/case_summaries/index.md" ]]; then
  echo "outside-work output candidate was committed" >&2
  exit 1
fi

"$FBT_BIN" review approve case_summaries --project-dir "$happy" --comment "conformance" >"$tmpdir/review-approve.txt"
grep -q "status: approved" "$tmpdir/review-approve.txt"
"$FBT_BIN" build --project-dir "$happy" --select weekly_support_insights >"$tmpdir/build-weekly.txt"
test -f "$happy/target/artifacts/support/weekly_insights.md"
"$FBT_BIN" docs generate --project-dir "$happy" >"$tmpdir/docs.txt"
test -f "$happy/target/docs/index.md"
if grep -R "$redaction_marker" "$happy/target/docs" >/dev/null; then
  echo "docs leaked redaction marker" >&2
  exit 1
fi

"$FBT_BIN" export openlineage --project-dir "$happy" --output "$tmpdir/openlineage.ndjson" >"$tmpdir/export-openlineage.txt"
"$FBT_BIN" export otel --project-dir "$happy" --output "$tmpdir/otel.json" >"$tmpdir/export-otel.txt"
"$FBT_BIN" export openlineage --project-dir "$happy" --output "$tmpdir/openlineage-again.ndjson" >"$tmpdir/export-openlineage-again.txt"
"$FBT_BIN" export otel --project-dir "$happy" --output "$tmpdir/otel-again.json" >"$tmpdir/export-otel-again.txt"
cmp "$tmpdir/openlineage.ndjson" "$tmpdir/openlineage-again.ndjson"
cmp "$tmpdir/otel.json" "$tmpdir/otel-again.json"
python3 - "$tmpdir/openlineage.ndjson" "$tmpdir/otel.json" "$redaction_marker" <<'PY'
import json
import re
import sys
from pathlib import Path

openlineage_path = Path(sys.argv[1])
otel_path = Path(sys.argv[2])
redaction_marker = sys.argv[3]

openlineage_text = openlineage_path.read_text()
otel_text = otel_path.read_text()
for label, text in [("openlineage", openlineage_text), ("otel", otel_text)]:
    if redaction_marker in text:
        raise SystemExit(f"{label} export leaked redaction marker")
    if "Login issue resolved" in text:
        raise SystemExit(f"{label} export leaked raw source content")

events = [json.loads(line) for line in openlineage_text.splitlines() if line.strip()]
if len(events) < 2:
    raise SystemExit("expected at least two OpenLineage events")
for event in events:
    if event.get("eventType") != "COMPLETE":
        raise SystemExit(f"unexpected OpenLineage event type: {event}")
    if event.get("schemaURL") != "https://openlineage.io/spec/1-0-0/OpenLineage.json#/definitions/RunEvent":
        raise SystemExit("OpenLineage event missing schemaURL")
    run_id = event.get("run", {}).get("runId", "")
    if not re.fullmatch(r"[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}", run_id):
        raise SystemExit(f"OpenLineage runId is not deterministic UUIDv5-shaped: {run_id}")
    outputs = event.get("outputs", [])
    if not outputs or "fbt_artifact" not in outputs[0].get("facets", {}):
        raise SystemExit("OpenLineage output missing fbt_artifact facet")
    for dataset in event.get("inputs", []) + outputs:
        for key, facet in dataset.get("facets", {}).items():
            if not key.startswith("fbt_"):
                raise SystemExit(f"custom OpenLineage facet lacks fbt_ prefix: {key}")
            if not str(facet.get("_schemaURL", "")).startswith("https://schemas.fbt.dev/openlineage/"):
                raise SystemExit(f"custom OpenLineage facet lacks immutable fbt schema URL: {key}")
if not any("fbt_approval" in output.get("facets", {}) for event in events for output in event.get("outputs", [])):
    raise SystemExit("OpenLineage export missing approval facet")

otel = json.loads(otel_text)
resource_spans = otel.get("resourceSpans", [])
if not resource_spans:
    raise SystemExit("OTel export missing resourceSpans")
spans = []
for resource_span in resource_spans:
    for scope_span in resource_span.get("scopeSpans", []):
        spans.extend(scope_span.get("spans", []))
if len(spans) < 2:
    raise SystemExit("expected OTel invocation and transform spans")

def attr_keys(span):
    return {attr.get("key") for attr in span.get("attributes", [])}

if not any("fbt.invocation.id" in attr_keys(span) for span in spans):
    raise SystemExit("OTel export missing invocation id attribute")
if not any("fbt.transform.id" in attr_keys(span) for span in spans):
    raise SystemExit("OTel export missing transform id attribute")
if not any("gen_ai.usage.input_tokens" in attr_keys(span) for span in spans):
    raise SystemExit("OTel export missing GenAI usage attribute")
if not any(span.get("events") for span in spans):
    raise SystemExit("OTel export missing runner span events")
PY

printf '\n- Dirty propagation fixture\n' >>"$happy/assets/support_style_guide.md"
"$FBT_BIN" plan --project-dir "$happy" --select case_summaries >"$tmpdir/plan-dirty.txt"
grep -Eq "run transform\\..*\\.case_summaries" "$tmpdir/plan-dirty.txt"

denied="$tmpdir/denied"
"$FBT_BIN" init "$denied" --template support >"$tmpdir/init-denied.txt"
cat >"$denied/policies/support.yml" <<'YAML'
policies:
  - name: support_agent_scope
    read: ["data/support/"]
    write: ["target/artifacts/other/"]
    network: false
YAML

set +e
"$FBT_BIN" build --project-dir "$denied" --select case_summaries >"$tmpdir/build-denied.txt" 2>"$tmpdir/build-denied.err"
denied_code=$?
set -e
if [[ "$denied_code" -eq 0 ]]; then
  echo "expected policy-denied build to fail" >&2
  exit 1
fi
if [[ -e "$denied/target/artifacts/support/case_summaries/index.md" ]]; then
  echo "policy-denied output was committed" >&2
  exit 1
fi
test -f "$denied/.fbt/state/policy_decisions.json"
grep -q '"status": "denied"' "$denied/.fbt/state/policy_decisions.json"

echo "conformance: ok"
