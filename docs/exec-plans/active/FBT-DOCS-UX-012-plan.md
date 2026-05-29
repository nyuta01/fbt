# FBT-DOCS-UX-012 Make manual-generation docs show concrete source-to-artifact evidence

## Observation

README now gives a clear source/instruction/runner/artifact story, but the
manual-generation docs and practical example READMEs still lean more toward
workflow commands. A first-time user can run the examples, but may still ask
which source content, format instruction, and runner behavior produced the
manual.

## Decision

Move the README-level concreteness into the practical docs. The manual examples
should show representative source lines, the relevant format/prompt assets, the
runner configuration, the generated artifact path, a short artifact excerpt,
and the inspection output that proves lineage.

## Permanent Fix

Update `apps/docs` manual-generation content, the incident/support example
READMEs, and the practical manual-generation reference doc. Keep examples real:
do not replace the external-runner flow with mock-only behavior.

## Next Check

Run practical examples smoke, docs-site build, and `make verify`.
