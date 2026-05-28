# FBT-UNIX-015 Converge Value Narrative On The Build Receipt

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make fbt's value obvious to a first-time user without requiring them to learn
every feature first.

## Observation

README and the CLI now mostly communicate a small tool, but deeper docs still
list runner, eval, policy, lineage, and state concepts side by side. That can
make fbt feel broader than it is. The clearest user value is the build receipt:
what changed, what ran, what version was produced, and how lineage can be
exported.

After comparing related tools, `build` remains the best primary execution verb:
dbt, Make, Bazel, and similar tools use build semantics for dependency-driven
outputs, while `generate` would overfit fbt to LLM text generation and `run`
would blur the boundary with external runners.

## Decision

Use one primary value sentence across README, docs, and examples:

```text
fbt gives generated files a build receipt.
```

Details such as runners, evals, policies, artifact versions, and standard
exports should support that sentence instead of competing with it.

Keep `build` as the command name and explain it in build-tool terms: selected
file inputs plus instructions and an external runner produce declared artifact
outputs plus a local receipt.

## Permanent Fix

- Normalized README, usage guide, CLI reference, design doc, docs site, and
  examples around `sources + instructions + runner -> artifact + build receipt`.
- Kept examples centered on `plan -> build -> artifact/diff/export`.
- Clarified that `plan` is read-only and `build` owns artifact production,
  checks, immutable versions, local state, and receipts.
- Updated CLI help so `build` reads as artifact construction rather than a raw
  runner invocation.

## Next Check

Run:

```sh
make validate-docs
make docs-site-build
make verify
```
