# FBT-DOCS-UX-010 Make README source-to-artifact examples concrete

## Observation

The README explained the fbt mental model, but the first-time reader still had
to infer which concrete files were sources, which files acted as instructions,
which runner command was called, and which artifact was produced.

## Decision

Added concrete source-to-artifact mapping to the README's offline support loop
and real incident runbook example. The README now names the source files,
source declarations, instruction assets, transform recipes, runners, artifacts,
checks, and receipt locations.

## Permanent Fix

Kept the example grounded in checked-in files and retained the 220-line README
guard so future README changes stay concise and verifiable.

## Next Check

Done. `validate_docs` and `make verify` pass.
