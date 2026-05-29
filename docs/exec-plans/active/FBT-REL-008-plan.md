# FBT-REL-008 Add One-Command CLI Installer

## Observation

The core CLI release assets exist, but first-time users still had to choose the
right archive manually, download `SHA256SUMS`, run checksum verification, unpack
the archive, and place the binary on `PATH`. That is acceptable for maintainers
but too much friction for a first successful loop.

## Decision

Add a small shell installer that uses the existing GitHub Release archives
instead of introducing a package registry, daemon, tap repository, or provider
dependency. The installer should detect the local OS/architecture, download the
matching archive, verify `SHA256SUMS`, install to a user-local bin directory,
and print the installed version.

## Permanent Fix

Add root `install.sh`, a local smoke test that exercises the installer against
a generated file:// release archive, and docs that make the one-command path
the first install option. Keep manual archive install and source builds as
fallbacks.

## Next Check

Installer checks:

```sh
make install-script-smoke
make verify
```
