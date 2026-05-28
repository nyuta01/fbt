# FBT-DOCS-UX-005 Expand README concrete example

## Observation

The README's concrete example still jumped from a directory listing directly to
commands. A first-time reader could not see what the input files contained,
what the transform declared, what output shape was required, or what fbt added
around the runner execution.

## Decision

Keep the README product-focused but make the concrete example self-explanatory:
show representative source content, response-log content, transform YAML,
required manual sections, output path, and fbt's responsibility for recording
source/assets/policy/eval/runner/model/version lineage.

## Permanent Fix

README now explains the support resolution manual example through file content
and transform logic before showing commands. The quickstart fixture remains
secondary and links to the full transcript.

## Next Check

```sh
make verify
```
