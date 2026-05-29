# FBT-SEC-001 Document OS Sandbox Execution Profiles For High-Security Use

## Observation

fbt policy, path containment, output boundaries, and adapter safety checks are
covered, but OS-level sandboxing is intentionally external to core. Users with
strict security requirements need concrete execution profiles.

The docs need to explain which guarantees fbt core owns and which controls must
be supplied by CI, containers, host policy, or runner/adaptor launch wrappers.

## Decision

Document how to run fbt and runner commands inside external isolation layers
such as CI sandboxes, containers, macOS sandboxing, Linux namespace/seccomp
profiles, or network-denied environments.

Keep the implementation boundary unchanged: no in-core sandbox manager,
privileged launcher, daemon, provider SDK, or agent runtime.

## Permanent Fix

Add a security guide that keeps the core boundary clear: fbt records and
enforces its own file/build contract, while OS-level process and network
isolation belongs to the execution environment.

Added `docs/security/os-sandbox-profiles.md`, a docs-site reference page,
security spec and runner-adapter references, and
`scripts/check-security-profiles.py` behind `make security-profiles-check` and
`make verify`.

## Next Check

`make security-profiles-check` and `make verify` passed.
