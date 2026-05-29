#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FBT_BIN="${FBT_BIN:-$ROOT_DIR/bin/fbt}"
if [[ ! -x "$FBT_BIN" ]]; then
  echo "FBT_BIN is not executable: $FBT_BIN" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

project="${FBT_STANDARD_BACKEND_PROJECT_DIR:-$tmpdir/fbt-viz-knowledge}"
if [[ -e "$project" ]]; then
  rm -rf "$project"
fi

"$FBT_BIN" init "$project" --template support >"$tmpdir/init.txt"
"$FBT_BIN" build --project-dir "$project" --select case_summaries >"$tmpdir/build-case.txt"
"$FBT_BIN" build --project-dir "$project" --select weekly_support_insights >"$tmpdir/build-weekly.txt"

lineage_dir="$project/target/lineage"
telemetry_dir="$project/target/telemetry"
mkdir -p "$lineage_dir" "$telemetry_dir"
openlineage_file="$lineage_dir/openlineage.ndjson"
otel_file="$telemetry_dir/otel.json"

"$FBT_BIN" export openlineage --project-dir "$project" --output "$openlineage_file" >"$tmpdir/export-openlineage.txt"
"$FBT_BIN" export otel --project-dir "$project" --output "$otel_file" >"$tmpdir/export-otel.txt"

grep -q '"eventType":"COMPLETE"' "$openlineage_file"
grep -q '"resourceSpans"' "$otel_file"
python3 -m json.tool "$otel_file" >/dev/null

marquez_status="skipped"
marquez_events=0
if [[ -n "${FBT_MARQUEZ_URL:-}" ]]; then
  marquez_endpoint="${FBT_MARQUEZ_URL%/}"
  if [[ "$marquez_endpoint" != */api/v1/lineage ]]; then
    marquez_endpoint="$marquez_endpoint/api/v1/lineage"
  fi
  while IFS= read -r event; do
    [[ -z "$event" ]] && continue
    http_code="$(curl -sS -o "$tmpdir/marquez-response.txt" -w '%{http_code}' \
      -X POST "$marquez_endpoint" \
      -H 'Content-Type: application/json' \
      --data "$event")"
    case "$http_code" in
      2*|3*) ;;
      *)
        echo "standard-backend-smoke: Marquez POST failed with HTTP $http_code" >&2
        cat "$tmpdir/marquez-response.txt" >&2 || true
        exit 1
        ;;
    esac
    marquez_events=$((marquez_events + 1))
  done <"$openlineage_file"
  marquez_status="posted"
fi

otlp_endpoint="${FBT_OTLP_TRACES_URL:-${OTEL_EXPORTER_OTLP_TRACES_ENDPOINT:-}}"
otlp_status="skipped"
if [[ -n "$otlp_endpoint" ]]; then
  http_code="$(curl -sS -o "$tmpdir/otlp-response.txt" -w '%{http_code}' \
    -X POST "$otlp_endpoint" \
    -H 'Content-Type: application/json' \
    --data-binary @"$otel_file")"
  case "$http_code" in
    2*|3*) ;;
    *)
      echo "standard-backend-smoke: OTLP POST failed with HTTP $http_code" >&2
      cat "$tmpdir/otlp-response.txt" >&2 || true
      exit 1
      ;;
  esac
  otlp_status="posted"
fi

if [[ -n "${FBT_STANDARD_EVIDENCE_DIR:-}" ]]; then
  mkdir -p "$FBT_STANDARD_EVIDENCE_DIR"
  cp "$openlineage_file" "$FBT_STANDARD_EVIDENCE_DIR/openlineage.ndjson"
  cp "$otel_file" "$FBT_STANDARD_EVIDENCE_DIR/otel.json"
  cat >"$FBT_STANDARD_EVIDENCE_DIR/smoke-summary.txt" <<TEXT
standard-backend-smoke
project: $project
openlineage_file: $openlineage_file
otel_file: $otel_file
marquez_status: $marquez_status
marquez_events: $marquez_events
otlp_status: $otlp_status
screenshot_rule: capture screenshots from Marquez, Jaeger, Tempo, Grafana, or OpenMetadata after ingestion; do not create a custom fbt graph image.
TEXT
fi

echo "standard-backend-smoke: exports ok"
echo "standard-backend-smoke: marquez $marquez_status ($marquez_events events)"
echo "standard-backend-smoke: otlp $otlp_status"
