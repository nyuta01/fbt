# Contributing

`fbt` is early-stage. Keep changes small, spec-backed, and easy for agents and
humans to verify.

## Development Loop

Start from the repository harness:

```sh
make agent-init
```

Before calling work complete:

```sh
make verify
```

`make verify` is the single required gate. It runs harness checks, documentation
checks, Go formatting checks, Go tests, and the CLI smoke test.

## Change Discipline

- Update the relevant spec or design document when behavior changes.
- Update `docs/exec-plans/feature-list.json` when task state changes.
- Keep `AGENTS.md` compact; route to source-of-truth docs instead of expanding it.
- Keep `fbt` core lightweight. Transform execution belongs to external runners.
- Prefer deterministic checks over one-off fixes.

## Commit Discipline

Use focused commits. Do not combine unrelated changes such as documentation,
refactors, feature work, and tooling changes in one commit.

Recommended categories:

- docs
- harness
- cli
- spec
- runner-protocol
- state
- security

## Local Requirements

- Go, as declared in `go.mod`
- Python 3.13 or compatible Python 3 for harness scripts
- `make`

No daemon, scheduler, metadata database, web server, or cloud account is required
for the base development loop.
