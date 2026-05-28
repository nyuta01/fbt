# Quality Score

Scores are lightweight retrospective signals for agent work. Use 1-5, where 5
means strong and mechanically protected.

| Domain | Score | Evidence | Weak Spot | Next Task |
|---|---:|---|---|---|
| Harness PDCA | 4 | `make verify` includes harness, drift, docs, Go, CLI smoke checks, and task-by-task commits | Product conformance tests are not implemented yet | `FBT-MVP-016` |
| fbt Spec Coverage | 5 | Core, project config, manifest, state, runner protocol, schema/versioning, runner discovery, security/conformance, usage, and example specs exist | Eval/review/security specs still need implementation evidence | `FBT-MVP-011` |
| Go CLI Scaffold | 4 | CLI now exposes help/version plus parse, plan, build, state, artifact, and runner diagnostics with tests and smoke coverage | Eval, review, diff, and docs commands are still pending | `FBT-MVP-012` |
| Runner Discovery | 4 | `internal/plugin`, `internal/runner`, and `fbt runner` tests cover project config, plugin manifests, PATH convention, missing commands, and diagnostics | Discovery validation is not yet backed by full protocol capability checks in CLI | `FBT-MVP-011` |
| Runner Protocol | 4 | `internal/protocol` fake-runner tests cover initialize, runTransform, JSONL notifications, output candidates, JSON-RPC errors, cancellation, and build invocation | Capability validation remains shallow | `FBT-MVP-011` |
| Local Runners | 4 | `runners/fake` and `runners/command` process-level tests verify protocol compatibility, output candidates, and fake-runner build use without external services | Command runner is not yet exercised through `fbt build` | `FBT-MVP-014` |
| Parser Baseline | 4 | `internal/project`, `internal/config`, and `internal/parser` now have tests for config versioning, artifact aliases, resource refs, path containment, and parse CLI wiring | Parser still emits first-pass diagnostics without schema-generated validation | `FBT-MVP-010` |
| Manifest Graph | 4 | `internal/manifest` and `internal/graph` now test canonical IDs, parent/child maps, deterministic JSON, selectors, CLI manifest writes, and build integration | Planner/build integration still needs fuller state comparison | `FBT-MVP-011` |
| Artifact Descriptors | 4 | `internal/artifact` and `internal/security` test file/directory descriptors, artifact version IDs, path containment, symlink rejection, and build commit use | Policy enforcement around candidate paths is still first-pass | `FBT-MVP-011` |
| Local State | 4 | `internal/state` tests atomic snapshots, append-only run results, locks, immutable artifact versions, CLI inspection, and build commits | Review/eval/policy state writers are not yet wired into lifecycle | `FBT-MVP-012` |
| Planner Baseline | 4 | `internal/planner` tests run/skip/blocked actions, dirty reasons, selected sets, review/confidence blocking, `fbt plan`, and clean build reruns | Dirty reasons remain first-pass until richer run records are produced | `FBT-MVP-011` |
| Security Boundaries | 4 | Security/conformance spec defines trust boundary, path rules, approval blocking, and fake-runner scenarios; path helper tests now cover escape and symlink rejection | No executable conformance suite yet | `FBT-MVP-011` |
