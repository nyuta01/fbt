# FBT-RUNNER-026 Harden Official Adapter Install And Live Verification UX

## Observation

The runner boundary is conceptually correct: fbt core stays provider-free and
calls external runner commands. In practice, user success depends heavily on
official adapters feeling installable, versioned, diagnosable, and verifiable.
If adapter setup fails, users experience it as fbt failing.

Updated observation: users now have an explicit verification ladder for
official adapters: fixture/fake conformance, clean VCS install smoke, installed
command smoke, and opt-in live build smoke.

## Decision

Keep adapters outside core but harden their user path. Official adapters should
have clear install commands, versioning expectations, conformance checks,
diagnostic behavior, and opt-in live smoke guidance for environments with
credentials.

## Permanent Fix

Review official adapter docs/scripts and fill gaps around install, `doctor`
diagnostics, conformance, fake-response checks, live opt-in smoke, and provider
or CLI-agent version expectations. Do not add provider SDKs or credential
storage to fbt core.

Implemented in:

- `Makefile` with `official-adapter-smoke`
- `docs/runner-adapters.md`
- `docs/runner-authoring-guide.md`
- `apps/docs/src/content/docs/runners/external-runners.mdx`
- `apps/docs/src/content/docs/runners/openai-runner.mdx`

## Next Check

Run adapter install smoke, runner conformance, docs scans for install/live
verification commands, and `make verify`.

Completed:

- `make official-adapter-smoke`
- docs scan for `official-adapter-smoke`, `adapter-install-smoke`,
  `runner-adapter-smoke`, and `FBT_RUNNER_ADAPTER_SMOKE_BUILD=1`
- `make verify`
- `make adapter-install-smoke` after committing, because the target requires a
  clean committed working tree
