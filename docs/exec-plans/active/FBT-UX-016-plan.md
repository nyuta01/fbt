# FBT-UX-016 Show Source-File Change Details In Dirty Explanations

## Observation

Daily source directories can accumulate many files. The previous dirty
explanation told users that a source changed, but not which concrete files were
added, changed, or removed.

## Decision

Expose source file-set deltas in `plan` and `artifact explain` without adding a
scheduler, watermark store, or partition engine. Source paths are recorded in
the existing manifest `files` map and planner nodes carry `source_changes` for
machine-readable output.

## Permanent Fix

Source-resolved files are now associated with their source resources in the
manifest. Planner source deltas report added, changed, and removed paths, and
the CLI prints those deltas in both plan and artifact explanation surfaces.

## Next Check

Done. Planner tests cover add/change/delete deltas, CLI tests cover plan and
artifact explain output after a successful build, and `make verify` passed.
