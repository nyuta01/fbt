# FBT-SEC-002 Require Explicit Policy For Agent Transforms

## Observation

Agent transforms currently produce an `AGENT_POLICY_MISSING` warning when policy
is omitted. The security spec already says this should become a parse error
before stable v1.

## Decision

Make explicit policy mandatory for `type: agent` transforms. Keep the change
limited to agent transforms so command and LLM transforms retain their current
default-policy behavior.

## Permanent Fix

Update parser behavior, diagnostics, examples, docs, and conformance so agent
transforms cannot accidentally run under an implicit policy.

## Next Check

Run parser tests, conformance negative cases, docs validation, and
`make verify`.

