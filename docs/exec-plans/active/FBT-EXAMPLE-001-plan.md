# FBT-EXAMPLE-001 Add practical manual generation examples

## Observation

The existing committed knowledge example is useful for local MVP smoke, but it
uses deterministic demo runners. Users evaluating fbt for operational manuals
need examples shaped like real business workflows with external runner
configuration, source evidence, output format assets, eval gates, and review.

## Decision

Add two production-shaped examples that parse and plan without provider calls
but require an external fbt-compatible runner for doctor/build. Cover incident
logs to incident runbook, and support inquiry/response logs to support
resolution manual.

## Permanent Fix

Added `examples/incident_response_runbook`,
`examples/support_resolution_manual`, and
`docs/examples/practical-manual-generation-examples.md`. Added
`make practical-examples-smoke` to parse and plan both examples in the default
verification gate without calling external providers.

## Next Check

Run:

```sh
make verify
```
