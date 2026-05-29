# FBT-SEC-002 Require Explicit Policy For Agent Transforms

## Observation

Agent transforms previously produced an `AGENT_POLICY_MISSING` warning when
policy was omitted, leaving high-risk external agent execution on an implicit
default.

## Decision

Make explicit policy mandatory for `type: agent` transforms and leave `llm` and
`command` default-policy behavior unchanged.

## Permanent Fix

Parser validation now emits `AGENT_POLICY_MISSING` as an error, docs state that
agent transforms require explicit policy, docs snippets include agent policies,
and conformance covers the negative parse case.

## Next Check

`make verify` must pass; future agent security improvements should stay in
adapter policy mapping, external OS sandbox profiles, or host/CI isolation
unless a spec explicitly expands core.
