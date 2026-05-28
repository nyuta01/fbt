# FBT-REL-001 Stamp MVP version and release metadata

## Observation

The CLI, manifest/build fallback version, runner initialize metadata, CLI smoke,
and dist check still used `0.0.0-dev`. Release builds also had no documented
source-level contract for stamping version, commit, and build date.

## Decision

Promote the MVP source default to `0.1.0`, keep `fbt version` stable for humans,
add JSON release metadata for automation, and make Makefile/dist builds stamp
`VERSION`, `COMMIT`, and `BUILD_DATE` via Go linker variables.

## Permanent Fix

Added `internal/version` as the single release metadata source with a source
default of `0.1.0` and linker-stamped `VERSION`, `COMMIT`, and `BUILD_DATE`.
Wired that metadata into `fbt version`, JSON version output, manifest/build
fallbacks, runner initialize metadata, `make build`, CLI smoke, dist check, and
the CLI reference.

## Next Check

Run:

```sh
make verify
```
