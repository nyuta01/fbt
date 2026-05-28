# FBT-UX-008 Align human CLI output columns

## Observation

After the Cobra migration, default CLI output was clearer but still visually
uneven. Item details mixed labels such as `because:`, `output:`, and `next:`
with different label widths, so values shifted between rows. `artifact explain`
dependency rows had a similar issue because status, role, resource, and details
were printed as free-form text.

Glamour was considered because it renders Markdown for terminal output, but the
problem here is structured status output rather than Markdown document
rendering. Adding a Markdown renderer would not directly solve row alignment and
would add a heavier dependency than needed.

## Decision

Keep default fbt output plain and deterministic. Use fixed-width key/value rows
and Go's `text/tabwriter` for dependency and output tables. Reserve richer
Markdown rendering for a future command that actually renders Markdown artifact
content or docs.

## Permanent Fix

Aligned summary, plan-node, build-run, artifact path/show, producer, detail,
dependency, and output sections with shared formatting helpers. Updated smoke
checks and docs examples to assert the aligned output shape.

## Next Check

Verified:

```sh
make verify
```

Result: passed.
