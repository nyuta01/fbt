#!/usr/bin/env sh
set -eu
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift || true
done
if [ -n "$out" ]; then
  printf '# Codex CLI Adapter Conformance\n' > "$out"
fi
printf '# Codex CLI Adapter Stdout\n'
