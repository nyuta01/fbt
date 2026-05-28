# FBT-UNIX-004 Add Existing-Tool Runner Examples For Remark And Pandoc

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Show that fbt composes with existing document tools instead of reimplementing
Markdown processing or document conversion in core.

## Observation

The docs claimed that remark and Pandoc should stay outside fbt, but there was
no runnable example showing how to wrap those tools as fbt-compatible command
transforms.

## Decision

Support a small `type: command` transform contract that passes an argv to an
external command runner. Use deterministic project-local wrappers for smoke,
and document where real `remark` and `pandoc` calls belong.

## Permanent Fix

- Added command-transform `command` config through parser, manifest, and build
  protocol payloads.
- Extended the command runner to advertise `pdf` output support and run wrapped
  commands from `FBT_COMMAND_WORKDIR` when set.
- Added `examples/markdown_toolchain` with remark-style Markdown normalization
  and Pandoc-style PDF conversion.
- Added practical smoke coverage for the new example.
- Updated runner authoring, usage, project config, docs-site, and examples
  guidance.

## Next Check

Run:

```sh
make verify
```

Expected result: practical examples smoke builds both command-tool artifacts
without adding remark or Pandoc to fbt core.
