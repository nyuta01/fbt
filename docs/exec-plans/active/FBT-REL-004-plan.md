# FBT-REL-004 Simplify End-User Install Path For Core And Adapters

## Observation

The MVP release artifacts and checksums exist, but the user path from "I want
to install fbt" to "I can run fbt plus the adapter I need" can still be
simpler. Source checkout works well for contributors, but ordinary users need
clear copy-paste install and verification commands.

Updated observation: install docs now separate core CLI installation from
optional adapter installation, and explain checksum verification, source
builds, adapter `go install`, version pinning, source smoke, installed smoke,
and live smoke.

## Decision

Treat install UX as a release/documentation problem, not as core runtime scope.
Improve the release docs for core and official adapters, checksum verification,
version pinning, and the expected relationship between fbt core versions,
adapter module versions, and protocol compatibility.

## Permanent Fix

Update install/release docs and any release helper scripts needed to make the
end-user path clear. Preserve the provider-free core boundary and keep adapter
installation out-of-band.

Implemented in:

- `README.md`
- `apps/docs/src/content/docs/get-started/installation.mdx`
- `apps/docs/src/content/docs/reference/release.mdx`
- `docs/release.md`

## Next Check

Run install docs scans, `make dist-check`, adapter install smoke where
applicable, and `make verify`.

Completed:

- install docs scan for `SHA256SUMS`, adapter `go install`, version pinning,
  `official-adapter-smoke`, `adapter-install-smoke`, and live smoke commands
- `make dist-check`
- `make verify`
- `make adapter-install-smoke` after committing, because the target requires a
  clean committed working tree
