# FBT-H-001 Install AI-first engineering harness baseline

## Observation

`fbt` had a strong English design/specification set but no repository-local
engineering harness. Agents lacked a compact entrypoint, structured task state,
single verification command, CI gate, or executable CLI target.

## Decision

Install a lightweight Go-oriented harness modeled on the Folio harness and the
Learn Harness Engineering five-subsystem framework:

- compact `AGENTS.md`
- structured task state in `docs/exec-plans/feature-list.json`
- self-PDCA and permanent-fix methodology docs
- `make verify` as the single verification gate
- Python harness scripts with no third-party dependencies
- minimal Go CLI scaffold with unit tests and smoke test
- GitHub Actions workflow that runs `make verify`

## Permanent Fix

The harness now rejects missing required files, broken local Markdown links,
Japanese suffixed docs, stale active plans, unlinked failure entries, Go format
drift, and CLI smoke regressions through `make verify`.

## Next Check

Run:

```sh
make verify
```

Schema/versioning, artifact descriptor registry, runner discovery, security
model, and conformance scenarios are now tracked by completed follow-up plans
`FBT-H-002` through `FBT-H-004`.
