# Standard Backend Evidence

This directory is intentionally empty in the repository.

When a local Marquez, Jaeger, Tempo, Grafana, or OpenMetadata environment is
available, run `make standard-backend-smoke` with `FBT_STANDARD_EVIDENCE_DIR`
pointing at a temporary evidence directory. Keep screenshots out of core docs
unless they were captured from a real standard backend after ingestion.

Example:

```sh
FBT_MARQUEZ_URL=http://localhost:5000 \
FBT_OTLP_TRACES_URL=http://localhost:4318/v1/traces \
FBT_STANDARD_EVIDENCE_DIR=/tmp/fbt-standard-evidence \
make standard-backend-smoke
```

The smoke target copies the OpenLineage and OTLP/JSON exports plus a
`smoke-summary.txt` file into the evidence directory.
