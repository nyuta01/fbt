# FBT-DOCS-UX-002 Clarify quickstart diagram intent

## Observation

The support-loop diagram introduced for the docs was neither a standard
lineage visualization nor an actual UI/output screenshot. It was an ad-hoc
diagram derived from the quickstart command sequence, and that made its purpose
unclear: users could not tell whether it was an architecture diagram, a concrete
run trace, or a lineage graph.

## Decision

Remove the custom quickstart diagram instead of trying to rescue it. The
quickstart is better explained through the actual command transcript,
generated file paths, artifact excerpts, and standard export commands.

## Permanent Fix

README, usage guide, and docs-site entry pages no longer show the custom
support-loop graphic. The docs now state that the quickstart evidence is the
CLI output and generated files, and graph visualization should come from
standard exports such as OpenLineage or OTel rather than an invented fbt image.

## Next Check

```sh
make verify
```
