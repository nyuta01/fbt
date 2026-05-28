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

happy="$tmpdir/happy"
"$FBT_BIN" init "$happy" --template support >"$tmpdir/init-happy.txt"
redaction_marker="FBT_CONFORMANCE_SECRET_DO_NOT_EXPORT"
printf '\n- Do not export marker: %s\n' "$redaction_marker" >>"$happy/assets/support_style_guide.md"
printf '{"id":"T-secret","summary":"%s","impact":"redaction fixture"}\n' "$redaction_marker" >>"$happy/data/support/tickets/2026-05-28.jsonl"
"$FBT_BIN" build --project-dir "$happy" --select case_summaries >"$tmpdir/build-case.txt"
test -f "$happy/target/artifacts/support/case_summaries/index.md"

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

"$FBT_BIN" review approve case_summaries --project-dir "$happy" --comment "conformance" >"$tmpdir/review-approve.txt"
grep -q "status: approved" "$tmpdir/review-approve.txt"
"$FBT_BIN" build --project-dir "$happy" --select weekly_support_insights >"$tmpdir/build-weekly.txt"
test -f "$happy/target/artifacts/support/weekly_insights.md"
"$FBT_BIN" docs generate --project-dir "$happy" >"$tmpdir/docs.txt"
test -f "$happy/target/docs/index.md"

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

echo "conformance: ok"
