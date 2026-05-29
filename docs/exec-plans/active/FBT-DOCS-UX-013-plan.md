# FBT-DOCS-UX-013 Strengthen runner and standards docs with runnable examples

## Observation

The runner and standards docs are accurate but thinner than README. Pages such
as external runners, authoring contract, OpenAI runner, lineage model, and
project config describe the concepts, but they do not always include enough
runnable commands, minimal config, expected output, and concrete file pointers
to stand alone for a first-time user.

## Decision

Keep the docs site concise, but make each thin page self-contained enough for a
user to execute or verify the concept. Runner pages should include install/use
patterns, conformance checks, and failure diagnostics. Standards pages should
show exact export commands, expected summaries, and concrete backend handoff
shape.

## Permanent Fix

Add runnable examples and short output excerpts to the docs-site runner,
lineage/standards, and project-config pages. Cross-link to the deeper repository
specs only after the page has answered the practical "how do I use this?"
question.

## Next Check

Run docs-site build and `make verify`; add deterministic docs scans if the new
examples introduce phrases that should remain guarded.
