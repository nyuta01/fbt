# FBT-RUNNER-021 Make official adapter modules remotely installable

## Observation

The official adapter design promises out-of-band installation such as
`go install github.com/nyuta01/fbt/adapters/openai/cmd/fbt-runner-openai@version`.
The current adapter modules depend on `github.com/nyuta01/fbt/sdk/go v0.0.0`
with a local `replace ../../sdk/go`, so remote `go install ...@main` fails
outside the source checkout.

## Decision

Keep the monorepo and nested adapter modules, but remove local SDK `replace`
directives from adapter `go.mod` files. Use `go.work` only for local
development. Adapter modules depend on a normal VCS-resolved `sdk/go` module
version, and release docs now require module-scoped tags for nested modules.

## Permanent Fix

Added `make adapter-install-smoke`, which clones the current committed
repository into a temporary bare VCS remote, disables `go.work`, rewrites the
GitHub URL to that bare clone, and verifies all official adapter commands can
be installed with `go install module@commit`.

## Next Check

Done. `GOWORK=off` adapter tests pass, `make verify` passes, and
`make adapter-install-smoke` passes against the committed tree.
