# FBT-ARTIFACT-001 Implement semantic descriptors for common document types

## Observation

Artifact versions had raw descriptors and a `semantic_descriptor` field, but
the build lifecycle did not populate semantic descriptors for common text and
Markdown artifacts. Users could inspect byte-level identity but not stable
semantic structure metadata.

## Decision

Implement first-pass descriptors without changing artifact version identity:
`text_normalized_v1` for normalized text and `markdown_ast_v1` for Markdown
heading/code-block structure. Store them on artifact versions and surface them
through artifact inspection and generated docs.

## Permanent Fix

Added semantic descriptor generation in `internal/artifact`, wired build commit
to persist semantic descriptors, and updated CLI/docs/smoke/conformance coverage.
Raw descriptor digests remain the only artifact version identity input.

## Next Check

Run:

```sh
make verify
```
