# FBT-RUNNER-007 Define optional provider adapter packaging outside core

## Observation

The protocol, discovery, and authoring docs make external runners possible, but
provider and CLI-agent integration packages still lacked naming, manifest, and
release conventions that keep SDKs and heavyweight runtimes outside fbt core.

## Decision

Define optional adapter package conventions in documentation only. Core should
continue resolving ordinary commands and plugin manifests without learning
provider-specific package managers, SDKs, credentials, or subcommands.

## Permanent Fix

Added `docs/runner-adapters.md` with package names, command conventions,
project config examples, plugin manifests, PATH behavior, CLI-agent adapter
requirements, versioning, and conformance checklist. Linked it from README,
project config, runner discovery, runner protocol, usage, and authoring docs.

## Next Check

Run:

```sh
make verify
```
