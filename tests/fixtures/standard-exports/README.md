# Standard Export Fixtures

The executable fixtures for standard exports are generated inside
`tests/conformance/run.sh` from the support template. They intentionally include
a redaction marker in source and asset files, then assert that:

- `fbt export openlineage` emits OpenLineage RunEvent NDJSON with fbt facets.
- `fbt export otel` emits OTLP/JSON resource spans and transform spans.
- Neither export includes raw source content or the redaction marker.

The generated payloads stay in the temporary conformance directory so the repo
does not pin volatile timestamps or run IDs.
