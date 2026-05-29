# FBT-UNIX-017 Capture Critical Product Gap Backlog

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Convert the current product audit into restartable implementation tasks that
preserve fbt's Unix-style boundary.

## Observation

The project now expresses the core value clearly: source files plus
instructions plus an external runner produce generated artifacts and local build
receipts. The audit found that the biggest remaining risks are not missing
product categories such as review, scheduling, or custom visualization. The
critical gaps are build-tool reliability details: a selected dependency graph
does not complete in one invocation, failed runs are not recorded strongly
enough, some config fields look active without real behavior, YAML typo
handling is too permissive, CLI-agent adapter safety is mostly a documented
contract, high-volume state retention is undefined, standard visualization lacks
opt-in backend evidence, and a few stale docs still describe removed behavior.

## Decision

Add focused backlog items instead of expanding fbt's product scope. The new
tasks keep the base tool local-first and service-free while prioritizing the
features that make fbt trustworthy as a file build tool.

## Permanent Fix

- Added P0 tasks for one-invocation DAG builds, failed-run receipts, inert
  config cleanup, strict YAML diagnostics, CLI-agent adapter safety, and stale
  core-boundary docs.
- Added P1 tasks for real runner adapter smoke coverage, state retention
  hygiene, and opt-in standard backend visualization evidence.
- Updated quality score, progress notes, and failure tracking so future agents
  can start from the backlog rather than chat history.

## Next Check

Run:

```sh
make verify
```

Expected result: all harness, docs, drift, test, smoke, conformance, docs-site,
and distribution checks pass with the new backlog entries.
