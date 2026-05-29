# FBT-RUNNER-013 Research official runner adapter package design

## Observation

The runner boundary is product-defining for fbt. The repository has examples
and protocol fixtures, but no researched design for what an officially
maintained runner package should mean across packaging, implementation,
security, release, and support.

## Decision

Researched established extension models across Go, HashiCorp tools, Git,
kubectl, dbt adapters, MCP, Codex, Claude Code, and the current fbt runner
specs. The recommendation is to keep fbt core provider-free and ship official
runners as external adapter packages with conformance, checksums, docs,
security policy, and version compatibility.

## Permanent Fix

Added `docs/research/official-runner-adapter-design-report.md` and linked it
from `docs/runner-adapters.md`. The report defines the official adapter
contract, package shape, first adapter priorities, implementation phases, and
operating model.

## Next Check

Done. `make verify` passes. The next implementation task should start with the
provider-free official adapter foundation and avoid adding provider SDKs or
agent runtimes to fbt core.
