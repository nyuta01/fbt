# FBT-EXAMPLES-UX-002 Add daily QA operations example

## Observation

The practical examples could make a first-time user infer that fbt expects
JSONL sources and built-in review gates as the default operating model. That is
not the product model: fbt tracks declared filesystem sources, and approval
belongs outside core.

The daily operations use case also needed a runnable example that starts from
multiple source directories and produces multiple artifacts from one transform,
without relying on provider credentials. The source paths must stay stable
across days; encoding a single date in the source path makes the example look
like a one-off batch instead of a repeatable daily command.

## Decision

Add an offline `daily_qa_ops` example that uses stable Markdown directory
sources for the current processing window, plus reference docs. The first
transform produces three artifacts under stable `latest` logical paths. A
second transform consumes those artifacts and the current manual to produce one
manual update.

Keep scheduling, partition management, and provider execution outside fbt core;
the example shows the file build control-plane loop only.

## Permanent Fix

`examples/daily_qa_ops` now demonstrates a daily batch workflow with plain
Markdown directory sources under `data/qa/inbox/`, multiple input sources,
multiple output artifacts under `target/artifacts/.../latest/`, deterministic
demo runners, and a promoted manual update artifact. The practical examples
smoke now runs the full daily QA path from a copied project, including build and
artifact history.

## Next Check

```sh
make verify
```
