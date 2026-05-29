# FBT-CONF-002 Move Product Conformance Scenarios From Shell To A Structured Harness

## Observation

`make conformance` works, but product-level conformance scenarios are still
mostly shell-script driven. Shell is fine for glue, but large behavioral
coverage becomes easier to maintain when fixtures, assertions, and failure
messages are structured.

The previous `tests/conformance/run.sh` mixed project setup, command execution,
assertions, JSON validation, and error reporting in one long shell file.

## Decision

Move or wrap product conformance scenarios in a structured harness while
preserving deterministic behavior and the existing `make conformance` entry
point.

Use Python for the product conformance harness because the scenarios already
need structured JSON assertions, named scenario execution, temporary projects,
and richer failure messages. Keep `run.sh` only as a compatibility wrapper.

## Permanent Fix

Introduce a structured scenario runner, migrate the highest-value shell checks
first, and keep the old shell entry point only as a thin wrapper or remove it
once parity is proven.

Added `tests/conformance/run.py` with named scenarios for config diagnostics,
strict YAML diagnostics, build lifecycle, standard exports, runner failure
receipts, dirty planning, and policy denial. `tests/conformance/run.sh` now
execs the Python harness, and `make conformance` calls it directly.

## Next Check

`make conformance`, `FBT_BIN="$PWD/bin/fbt" bash tests/conformance/run.sh`, and
`make verify` passed.
