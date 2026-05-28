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
