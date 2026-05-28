# Quality Score

Scores are lightweight retrospective signals for agent work. Use 1-5, where 5
means strong and mechanically protected.

| Domain | Score | Evidence | Weak Spot | Next Task |
|---|---:|---|---|---|
| Harness PDCA | 4 | `make verify` includes harness, drift, docs, Go, and CLI smoke checks; active plans now cover completed P0 setup tasks | Product conformance tests are not implemented yet | Product parser task |
| fbt Spec Coverage | 5 | Core, project config, manifest, state, runner protocol, schema/versioning, runner discovery, security/conformance, usage, and example specs exist | Specs are still draft until implementation tests exercise them | Product parser task |
| Go CLI Scaffold | 3 | Minimal CLI, unit tests, and smoke test exist | Product commands are intentionally not implemented yet | Product parser task |
| Security Boundaries | 4 | Security/conformance spec defines trust boundary, path rules, approval blocking, and fake-runner scenarios | No executable conformance suite yet | Conformance harness task |
