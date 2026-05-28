# FBT-MVP-014 Implement init templates and runnable knowledge-loop example

## Observation

The CLI can parse, plan, build, eval, and review existing projects, but users
still need to hand-create project files. The repository also lacks a committed
example project that exercises the local build and review loop end to end.

## Decision

Implement:

- `internal/templates` for `blank`, `support`, `knowledge_ops`, and `incident`
  project scaffolds
- `fbt init [PROJECT_NAME] --template NAME [--force]`
- local runner wrapper scripts in generated runnable templates
- a committed `examples/knowledge_ops` project
- an e2e smoke that initializes, builds, reviews, and builds a downstream
  artifact without external services

## Permanent Fix

Added template parser tests, CLI init tests, a committed local knowledge-loop
example, and `scripts/smoke-knowledge-loop.sh` behind `make e2e-smoke` and
`make verify`.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
