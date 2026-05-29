# FBT-RUNNER-027 Plan

## Task

Harden the production runner reliability contract.

## Observation

The real-adapter daily pilot proved that official adapters can run the
production loop, but the production expectations were spread across adapter
code, protocol docs, security docs, and prior task notes.

## Decision

Add a single production reliability contract for runners, reference it from the
protocol and adapter docs, and add a deterministic repository check for the
contract and official adapter code markers.

## Permanent Fix

`make verify` now includes `runner-production-reliability-check`, which checks
that the reliability contract, protocol docs, adapter docs, conformance docs,
and official adapter source markers stay present.

## Next Check

- `make runner-production-reliability-check`
- `make production-pilot-smoke`
- `make verify`
