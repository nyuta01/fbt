# Contributing

`fbt` is early-stage. Keep changes small, spec-backed, and easy for agents and
humans to verify.

## Development Loop

Start from the repository harness:

```sh
make agent-init
```

Before calling work complete:

```sh
make verify
```

`make verify` is the single required gate. It runs harness checks, documentation
checks, Go formatting checks, Go tests, and the CLI smoke test.

## Change Discipline

- Update the relevant spec or design document when behavior changes.
- Update `docs/exec-plans/feature-list.json` when task state changes.
- Keep `AGENTS.md` compact; route to source-of-truth docs instead of expanding it.
- Keep `fbt` core lightweight. Transform execution belongs to external runners.
- Prefer deterministic checks over one-off fixes.

## Commit Discipline

Use focused commits. Do not combine unrelated changes such as documentation,
refactors, feature work, and tooling changes in one commit.

Recommended categories:

- docs
- harness
- cli
- spec
- runner-protocol
- state
- security

## Release Baseline

Maintainers publish releases from a clean tree after `make verify` passes.
Release builds use the version metadata contract documented in
`docs/cli-reference.md`: `VERSION`, `COMMIT`, and `BUILD_DATE` are stamped at
build time.

Before the first public MVP release, a maintainer must configure the GitHub
remote and choose the signing policy:

```sh
git remote add origin git@github.com:nyuta01/fbt.git
git config commit.gpgsign true
git config user.signingkey <KEY_ID>
```

The default policy is to keep existing local history intact and sign commits
and tags from the first public release point forward. If history is rewritten to
retroactively sign earlier commits, do that before the public remote becomes
the source of truth.

After verification, create and push the signed MVP tag:

```sh
make verify
git tag -s v0.1.0 -m "fbt v0.1.0"
git push -u origin main
git push origin v0.1.0
```

Check release integrity before announcing the release:

```sh
git remote -v
git log --show-signature -1
git tag -v v0.1.0
```

## Local Requirements

- Go, as declared in `go.mod`
- Python 3.13 or compatible Python 3 for harness scripts
- `make`

No daemon, scheduler, metadata database, web server, or cloud account is required
for the base development loop.
