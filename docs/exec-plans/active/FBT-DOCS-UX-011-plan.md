# FBT-DOCS-UX-011 Show README input-to-output content snippets

## Observation

The README named concrete source files, instruction files, runners, and artifact
paths, but a first-time reader still could ask what content was transformed
into what generated content.

## Decision

Changed the README examples from file-role mapping alone to input/output
snippets: the offline support example now shows the source ticket record and
generated Markdown artifact excerpt, and the incident example shows evidence
excerpts plus the procedure-style artifact shape.

## Permanent Fix

Kept the snippets grounded in checked-in example files and within the 220-line
README guard.

## Next Check

Done. `validate_docs` and `make verify` pass.
