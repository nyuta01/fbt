# FBT-CONF-001 Expand conformance beyond compact shell smoke

## Observation

The conformance target covered the main support loop and policy denial, but it
did not exercise several scenario classes listed in the security/conformance
spec, such as schema failures, clean reruns, docs redaction, and dirty-state
propagation.

## Decision

Keep the suite lightweight and shell-based for now, but expand the scenario
matrix inside `tests/conformance/run.sh` with deterministic checks that do not
require external services or providers.

## Permanent Fix

Added conformance checks for missing and unsupported `config_version`, clean
rerun skip behavior, docs redaction, standard export determinism/redaction, and
dirty propagation after a transform asset changes. Updated the conformance spec
coverage list to match the executable gate.

## Next Check

Run:

```sh
make verify
```
