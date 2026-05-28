# Quality Score

Scores are lightweight retrospective signals for agent work. Use 1-5, where 5
means strong and mechanically protected.

| Domain | Score | Evidence | Weak Spot | Next Task |
|---|---:|---|---|---|
| Harness PDCA | 4 | `make verify` includes harness, drift, docs, Go, and CLI smoke checks; MVP work is now registered as structured tasks | Product conformance tests are not implemented yet | `FBT-MVP-001` |
| fbt Spec Coverage | 5 | Core, project config, manifest, state, runner protocol, schema/versioning, runner discovery, security/conformance, usage, and example specs exist | Most specs are still draft until implementation tests exercise them | `FBT-MVP-002` |
| Go CLI Scaffold | 4 | CLI now exposes help/version plus parse, plan, state, artifact, and runner diagnostics with tests and smoke coverage | Build, eval, review, diff, and docs commands are still pending | `FBT-MVP-010` |
| Runner Discovery | 4 | `internal/plugin`, `internal/runner`, and `fbt runner` tests cover project config, plugin manifests, PATH convention, missing commands, and diagnostics | Protocol initialize/validation is not implemented until the protocol client task | `FBT-MVP-008` |
| Parser Baseline | 4 | `internal/project`, `internal/config`, and `internal/parser` now have tests for config versioning, artifact aliases, resource refs, and path containment | Parser still has no CLI surface until the parse command lands | `FBT-MVP-006` |
| Manifest Graph | 4 | `internal/manifest` and `internal/graph` now test canonical IDs, parent/child maps, deterministic JSON, selectors, and CLI manifest writes | Planner/build integration still needs fuller state comparison | `FBT-MVP-010` |
| Artifact Descriptors | 4 | `internal/artifact` and `internal/security` test file/directory descriptors, artifact version IDs, path containment, and symlink rejection | Descriptor code is not yet wired into build/state commit | `FBT-MVP-010` |
| Local State | 4 | `internal/state` tests atomic snapshot writes, append-only run results, lock behavior, stale lock recovery, immutable artifact version records, and CLI inspection | State is not yet integrated into build commits | `FBT-MVP-010` |
| Planner Baseline | 4 | `internal/planner` tests run/skip/blocked actions, dirty reasons, selected sets, review/confidence blocking, and `fbt plan` smoke coverage | Dirty reasons are still first-pass until build/run records are produced | `FBT-MVP-010` |
| Security Boundaries | 4 | Security/conformance spec defines trust boundary, path rules, approval blocking, and fake-runner scenarios; path helper tests now cover escape and symlink rejection | No executable conformance suite yet | `FBT-MVP-011` |
