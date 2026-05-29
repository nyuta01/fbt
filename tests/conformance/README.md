# Product Conformance Harness

`tests/conformance/run.py` is the structured product conformance harness for
fbt core behavior. It runs deterministic black-box scenarios through the built
CLI and reports failures with the scenario name plus the failed assertion.

`tests/conformance/run.sh` remains as a thin compatibility wrapper for agents
or scripts that still call the old shell entry point.

Run:

```sh
make conformance
```

The harness covers:

- project config version diagnostics
- strict YAML unknown-field diagnostics
- dependency-ordered builds and skipped clean builds
- artifact inspection and retention output
- failed-run receipts for runner capability, output-candidate, and policy
  failures
- deterministic OpenLineage and OTel exports with redaction checks
- dirty planning when an asset changes

