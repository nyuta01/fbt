# Markdown Toolchain Example

This example shows fbt composing with existing Unix-style document tools instead
of reimplementing them.

```text
raw Markdown directory
  -> remark-style normalized Markdown directory
  -> Pandoc-style PDF artifact
  -> fbt receipt with versions, checks, and lineage
```

The checked-in wrappers are deterministic so the example works offline. In a
real project, keep the same fbt project shape and replace the wrapper bodies
with calls to `remark` and `pandoc`.

## Run It

Preview the two-stage graph:

```sh
fbt plan --project-dir examples/markdown_toolchain --select tag:document_toolchain
```

Build the normalized Markdown artifact:

```sh
fbt build --project-dir examples/markdown_toolchain --select remark_markdown
```

Build the downstream document artifact:

```sh
fbt build --project-dir examples/markdown_toolchain --select pandoc_handbook
```

Inspect the receipt:

```sh
fbt artifact show handbook_pdf --project-dir examples/markdown_toolchain
fbt artifact explain handbook_pdf --project-dir examples/markdown_toolchain
```

## Replace The Wrappers

- `bin/run-remark-normalize` is where a real project would call `remark`.
- `bin/run-pandoc-handbook` is where a real project would call `pandoc`.
- fbt only records the artifact lifecycle. It does not parse Markdown ASTs or
  convert documents in core.
