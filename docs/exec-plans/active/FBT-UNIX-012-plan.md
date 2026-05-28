# FBT-UNIX-012 Make CLI Argument Handling Strict

Status: done  
Owner: agent  
Updated: 2026-05-29

## Goal

Make fbt safer as a small Unix-style CLI by refusing ambiguous or misspelled
input instead of silently broadening the operation.

## Observation

`fbt plan --bogus` and `fbt plan --help` were accepted as if no extra argument
had been supplied. `fbt build --select no_such` could fall through to a broader
build because the selector matched no transforms. Build selection also lagged
behind plan selection and did not support the same selector forms.

## Decision

Keep the command surface small and strict. Every command should reject unknown
flags and extra positional arguments. `--select` must match at least one
transform. `plan` and `build` should share selector semantics.

## Permanent Fix

- Added CLI argument count validation for commands and subcommands.
- Rejected `--select` on commands that do not use selection.
- Unknown per-command flags now fail with exit code `2`.
- Extra positional arguments now fail with exit code `2`.
- Empty transform selections now fail instead of selecting everything.
- `build --select` now supports `name`, `tag:`, `path:`, `resource_type:`, and
  `selector:` forms consistently with `plan`.
- Updated CLI docs and docs-site guidance to make `doctor -> plan -> build` the
  primary loop. `FBT-UNIX-013` later removed the old debug command surface.

## Next Check

Run:

```sh
make verify
```

Expected result: all checks pass and CLI typo cases fail fast.
