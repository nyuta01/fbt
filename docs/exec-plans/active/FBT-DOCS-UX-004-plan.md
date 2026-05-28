# FBT-DOCS-UX-004 Clarify README product story

## Observation

The README contained many accurate details, but it did not quickly answer the
first-time reader's questions: what fbt is for, what problem it solves, what it
can be used for, and how someone would use it in a real workflow.

## Decision

Rewrite the README as a product entry point instead of a documentation map.
Lead with the purpose, concrete operational use cases, a realistic support
manual example, what fbt owns in that workflow, what is available today, the
small quickstart fixture, project structure, boundaries, install, and docs.

## Permanent Fix

README now explains fbt as a local-first file build tool for turning
operational evidence into versioned, traceable knowledge artifacts. It uses the
support resolution manual example to show the intended workflow before listing
commands and reference docs.

## Next Check

```sh
make verify
```
