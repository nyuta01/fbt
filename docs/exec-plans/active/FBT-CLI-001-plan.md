# FBT-CLI-001 Decide and close run/debug command surface

## Observation

The CLI help and docs still exposed `run` and `debug` as planned-but-unimplemented
commands. That made the MVP command surface look broader than the behavior
users can actually rely on.

## Decision

Keep `build` as the execution command and `doctor`/`parse`/`state`/`runner
doctor` as the diagnostics surface. Remove `run` and `debug` placeholders from
help and CLI docs instead of implementing thin aliases.

## Permanent Fix

Removed planned-command handling from the CLI, removed `run`/`debug` sections
from the CLI reference, and updated smoke/tests so former placeholder commands
are treated as unknown commands.

## Next Check

Run:

```sh
make verify
```
