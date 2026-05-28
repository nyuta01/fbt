# FBT-EXAMPLES-UX-002 Add daily QA operations example

## Observation

The practical examples could make a first-time user infer that fbt expects
JSONL sources and review gates as the default operating model. That is not the
product model: fbt can track plain directories and review should be applied at
workflow boundaries where a team needs approval, not to every intermediate
artifact.

The daily operations use case also needed a runnable example that starts from
multiple source directories and produces multiple artifacts from one transform,
without relying on provider credentials. The source paths must stay stable
across days; encoding a single date in the source path makes the example look
like a one-off batch instead of a repeatable daily command.

## Decision

Add an offline `daily_qa_ops` example that uses stable Markdown directory
sources for the current processing window, plus reference docs. The first
transform produces three candidate artifacts without review under stable
`latest` logical paths. A second promotion transform consumes the candidate
artifacts and the current manual to produce one reviewed manual update.

Keep scheduling, partition management, and provider execution outside fbt core;
the example shows the file build control-plane loop only.

## Permanent Fix

`examples/daily_qa_ops` now demonstrates a daily batch workflow with plain
Markdown directory sources under `data/qa/inbox/`, multiple input sources,
multiple output artifacts under `target/artifacts/.../latest/`, deterministic
demo runners, candidate artifacts with `approval_status: not_required`, and a
promoted artifact that moves from pending review to approved. The practical
examples smoke now runs the full daily QA path from a copied project, including
build, artifact history, review show, and review approve.

## Next Check

```sh
make verify
```
