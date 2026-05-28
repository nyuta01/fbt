# FBT-UX-007 Improve CLI command framework and human output

## Observation

User-facing CLI output was too implementation-oriented. `plan`, `build`, and
`artifact` printed full internal IDs, raw lower-case states, and dense
key/value lines before showing the user's main answers: what will run, what was
created, where the artifact is, and what to do next.

The command parser was also hand-written, which made help text and flag
behavior harder to keep consistent as the CLI grew.

## Decision

Use Cobra for the command tree, global flags, per-command flags, help output,
and strict argument handling. Keep fbt-specific JSON output and command
handlers intact. Redesign default human output around short names, aligned
status labels, summary counts, and detail sections. Keep full resource IDs in
`Details` and in `--json` for automation.

## Permanent Fix

Added Cobra as the CLI command framework, migrated public commands into a
single command tree, and updated smoke/conformance checks to assert the new
human-readable output shape. Plan/build/artifact output now presents user-level
names first and reserves full IDs for detail sections.

## Next Check

Verified:

```sh
make verify
```

Result: passed.
