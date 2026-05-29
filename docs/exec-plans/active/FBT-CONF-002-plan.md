# FBT-CONF-002 Move Product Conformance Scenarios From Shell To A Structured Harness

## Observation

`make conformance` works, but product-level conformance scenarios are still
mostly shell-script driven. Shell is fine for glue, but large behavioral
coverage becomes easier to maintain when fixtures, assertions, and failure
messages are structured.

## Decision

Move or wrap product conformance scenarios in a structured harness while
preserving deterministic behavior and the existing `make conformance` entry
point.

## Permanent Fix

Introduce a structured scenario runner, migrate the highest-value shell checks
first, and keep the old shell entry point only as a thin wrapper or remove it
once parity is proven.

## Next Check

Run `make conformance` and `make verify`; failures should identify the scenario
and assertion that failed.
