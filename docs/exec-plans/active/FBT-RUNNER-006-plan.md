# FBT-RUNNER-006 Clarify demo runners and external runner UX

## Observation

The bundled deterministic LLM and agent runners were named `local.*` in
templates and generated wrappers. Users could reasonably confuse them with
first-class local provider runners instead of demo protocol fixtures.

## Decision

Make the generated runner identity visibly demo-only. Templates should use
`demo.llm`, `demo.agent`, and `bin/fbt-demo-*-runner`, while documentation
should show the shortest path from demo wrappers to external runner commands.

## Permanent Fix

Renamed generated template runner entries and the checked-in knowledge example
to `demo.*`, updated demo runner initialize/provenance names, added CLI init
guidance, documented replacement steps, and added template test coverage for
the demo naming.

## Next Check

Run:

```sh
make verify
```
