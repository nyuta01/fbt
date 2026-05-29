# FBT-RUNNER-026 Harden Official Adapter Install And Live Verification UX

## Observation

The runner boundary is conceptually correct: fbt core stays provider-free and
calls external runner commands. In practice, user success depends heavily on
official adapters feeling installable, versioned, diagnosable, and verifiable.
If adapter setup fails, users experience it as fbt failing.

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

## Next Check

Run adapter install smoke, runner conformance, docs scans for install/live
verification commands, and `make verify`.
