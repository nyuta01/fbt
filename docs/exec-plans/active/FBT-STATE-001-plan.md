# FBT-STATE-001 Emit full policy decision records during lifecycle

## Observation

The state schema included `policy_decisions.json`, but build lifecycle only used
policy checks inline to allow or deny commits. Allowed and denied outcomes were
not persisted consistently, so docs and audit state could not explain policy
decisions after the fact.

## Decision

Persist policy decisions during transform commit checks for both allowed and
denied outcomes. Record the policy ID, transform/run IDs, candidate artifact
version ID, status, check names/statuses/messages, and decision time.

## Permanent Fix

Added state read/put APIs for policy decisions, recorded decisions from build
before commit or denial return, included policy decision IDs in run results and
OTel attributes, and surfaced policy decisions in standard exports. Tests and
conformance now verify allowed and denied decisions are written.

## Next Check

Run:

```sh
make verify
```
