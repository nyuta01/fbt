# FBT-REL-004 Simplify End-User Install Path For Core And Adapters

## Observation

The MVP release artifacts and checksums exist, but the user path from "I want
to install fbt" to "I can run fbt plus the adapter I need" can still be
simpler. Source checkout works well for contributors, but ordinary users need
clear copy-paste install and verification commands.

## Decision

Treat install UX as a release/documentation problem, not as core runtime scope.
Improve the release docs for core and official adapters, checksum verification,
version pinning, and the expected relationship between fbt core versions,
adapter module versions, and protocol compatibility.

## Permanent Fix

Update install/release docs and any release helper scripts needed to make the
end-user path clear. Preserve the provider-free core boundary and keep adapter
installation out-of-band.

## Next Check

Run install docs scans, `make dist-check`, adapter install smoke where
applicable, and `make verify`.
