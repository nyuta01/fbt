# FBT-DOCS-UX-012 Make manual-generation docs show concrete source-to-artifact evidence

## Observation

README now gives a clear source/instruction/runner/artifact story, but the
manual-generation docs and practical example READMEs still lean more toward
workflow commands. A first-time user can run the examples, but may still ask
which source content, format instruction, and runner behavior produced the
manual.

Updated observation: the manual-generation docs now show the concrete
source-to-artifact chain for both practical examples. Each example includes
representative source records, response/product evidence, prompt and format
assets, runner name, output path, expected artifact shape, and `artifact
explain` receipt excerpts.

## Decision

Move the README-level concreteness into the practical docs. The manual examples
should show representative source lines, the relevant format/prompt assets, the
runner configuration, the generated artifact path, a short artifact excerpt,
and the inspection output that proves lineage.

## Permanent Fix

Update `apps/docs` manual-generation content, the incident/support example
READMEs, and the practical manual-generation reference doc. Keep examples real:
do not replace the external-runner flow with mock-only behavior.

Implemented in:

- `apps/docs/src/content/docs/get-started/manual-generation.mdx`
- `examples/incident_response_runbook/README.md`
- `examples/support_resolution_manual/README.md`
- `docs/examples/practical-manual-generation-examples.md`

## Next Check

Run practical examples smoke, docs-site build, and `make verify`.

Completed:

- `rg -n "Representative source evidence|Representative input|Concrete source-to-artifact mapping|Expected artifact shape|artifact explain|Source Evidence" ...`
- `make practical-examples-smoke`
- `npm --prefix apps/docs run build`
- `make verify`
