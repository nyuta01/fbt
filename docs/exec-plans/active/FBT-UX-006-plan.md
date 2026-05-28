# FBT-UX-006 Improve project validation and YAML authoring diagnostics

## Observation

Parser diagnostics had stable codes, but human output did not include line
numbers or actionable hints, which made YAML authoring errors slower to fix.

## Decision

Keep diagnostics lightweight in core. Add line and hint fields to parser
diagnostics, derive resource line numbers from YAML nodes where possible, and
print `file:line` plus `hint:` in CLI parse errors.

## Permanent Fix

Added diagnostic JSON tags, `line`, and `hint`; indexed resource YAML name lines
for sources, source artifacts, transforms, assets, policies, and evals; added
common remediation hints; updated parse output docs; and covered line/hint
behavior in parser and CLI tests.

## Next Check

Run:

```sh
make verify
```
