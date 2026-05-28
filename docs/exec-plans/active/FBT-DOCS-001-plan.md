# FBT-DOCS-001 Update documentation from draft to MVP-ready status

## Observation

The top-level and user-facing docs still described the project as draft or
target UX, and several CLI reference examples included reserved flags or
commands that the MVP CLI does not accept.

## Decision

Make the docs describe the implemented local MVP first: source-checkout quick
start, implemented command surface, local support knowledge loop, release
version metadata, and explicit non-goals/limitations. Keep larger examples as
extension patterns rather than the required MVP path.

## Permanent Fix

Updated `README.md`, `docs/usage-guide.md`, `docs/cli-reference.md`, and
`docs/examples/knowledge-loop-example.md` to reflect the runnable local support
loop, implemented CLI flags/commands/selectors, current docs output, and
maintainer-owned release publication blocker.

## Next Check

Run:

```sh
make verify
```
