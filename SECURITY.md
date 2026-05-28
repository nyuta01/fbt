# Security Policy

`fbt` is currently a draft project. Do not use it as a security boundary for
untrusted runners, untrusted documents, or untrusted agents until the security
and conformance suite is implemented.

## Reporting

If GitHub private vulnerability reporting is enabled for this repository, use
that path. Otherwise, contact the maintainers privately before opening a public
issue for vulnerabilities involving secret exposure, arbitrary file writes,
runner execution, path traversal, policy bypass, or supply-chain risk.

## Security Model

The base tool is local-first and delegates transform execution to external
runners. Core is responsible for project parsing, graph planning, policy
decisions, scoped work directories, artifact descriptors, immutable artifact
versions, approval state, and official commits.

External runners are trusted executables for MVP purposes. Core must still
validate runner protocol output, reject output candidates outside scoped work
directories, avoid storing secrets by default, and prevent failed or interrupted
runs from updating official artifact pointers.

See `docs/security-and-conformance-spec.md` for the full draft model.
