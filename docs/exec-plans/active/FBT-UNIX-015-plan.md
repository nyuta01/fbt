# FBT-UNIX-015 Converge Value Narrative On The Build Receipt

Status: todo  
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

## Decision

Use one primary value sentence across README, docs, and examples:

```text
fbt gives generated files a build receipt.
```

Details such as runners, evals, policies, artifact versions, and standard
exports should support that sentence instead of competing with it.

## Permanent Fix

Planned:

- Normalize the opening value proposition across README, usage guide, design
  doc, docs site introduction, and examples.
- Keep examples centered on `plan -> build -> artifact/diff/export`.
- Move runner/protocol/eval details behind the mental model instead of leading
  with them.

## Next Check

Run:

```sh
make validate-docs
make docs-site-build
make verify
```
