# FBT-DOCS-UX-001 Ground docs in actual user workflow

## Observation

The README and docs site looked like a polished project shell but did not
answer the practical user questions first: what fbt can do today, how the
commands run, what files are produced, what the generated artifacts look like,
and where the promised lineage graph and standard export records appear.
While capturing the quickstart from outside the repository, generated demo
runner wrappers failed because they invoked `go run ./runners/...` without
first changing to the source checkout.

## Decision

Fix the runner wrapper portability issue and rewrite the entry docs around a
captured support knowledge loop: exact commands, expected output, generated
files, artifact excerpts, artifact inspection, lineage commands, standard export
results, and diagrams.

## Permanent Fix

The support template now emits wrappers that `cd` to the source checkout before
running bundled demo runners, and the e2e smoke now runs an installed test
binary from a temporary directory with `doctor` included. README, source usage
docs, and the Starlight docs now show implemented workflows, actual command
outputs, generated artifact paths, and graph images for the support loop and
standards export path.

## Next Check

```sh
make verify
```
