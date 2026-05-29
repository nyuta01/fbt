# OS Sandbox Execution Profiles

Status: MVP-ready
Updated: 2026-05-29
Audience: teams running fbt with strict process, filesystem, or network
isolation requirements

fbt core does not implement an OS sandbox. fbt records and enforces its own
file/build contract: source descriptors, runner policy, output-candidate
containment, artifact commits, failed-run receipts, and standard exports.

Process isolation belongs to the environment that launches fbt and the runner
commands. Use these profiles when the runner is an LLM provider adapter, CLI
agent, document converter, OCR tool, or internal executable with a larger trust
surface than fbt core.

## What fbt Enforces

fbt core enforces:

- official artifact commits only through fbt
- output candidates must be under the invocation `work.outputs`
- logical artifact paths must stay under `artifact_path`
- source writes are not part of the official commit path
- output-size and descriptor checks before commit
- failed, denied, or interrupted runs do not advance current artifact pointers
- runner stderr diagnostics are bounded and redacted

fbt core does not enforce:

- process syscall restrictions
- kernel namespace isolation
- filesystem mount permissions
- host firewall rules
- provider account permissions
- terminal or browser access by external agent CLIs

## Profile 1: Local Trusted Runner

Use for development with deterministic demo runners or trusted internal command
runners.

```sh
fbt doctor --project-dir .
fbt plan --project-dir . --select tag:daily
fbt build --project-dir . --select tag:daily
```

Controls:

- run from a normal user account
- keep credentials in the shell or secret manager, not project files
- inspect `fbt doctor` before build
- inspect `fbt artifact explain` after build

This profile does not provide OS isolation.

## Profile 2: CI Sandbox

Use for repeatable builds where CI already provides an ephemeral checkout.

Controls:

- run in a short-lived CI job
- provide only the runner credentials needed by selected transforms
- avoid broad repository write tokens
- upload `target/` and `.fbt/` only when the downstream workflow needs them
- run `make verify`, `fbt plan`, and `fbt build` as separate visible steps

Example shape:

```sh
fbt doctor --project-dir "$GITHUB_WORKSPACE"
fbt plan --project-dir "$GITHUB_WORKSPACE" --select tag:publishable
fbt build --project-dir "$GITHUB_WORKSPACE" --select tag:publishable
fbt export openlineage --project-dir "$GITHUB_WORKSPACE" --output openlineage.ndjson
```

CI isolation is strongest when the job has no deploy token and no provider
credential unless the selected runner actually needs one.

## Profile 3: Container With Read-Only Sources

Use when you want filesystem mount boundaries around fbt and its runner.

Mount source, instruction, policy, and eval paths read-only. Mount `target/`
and `.fbt/` writable because fbt must write current artifacts, work
directories, immutable artifact versions, and receipts.

Example sketch:

```sh
docker run --rm --network=none \
  --read-only \
  --tmpfs /tmp \
  -v "$PWD/fs_project.yml:/workspace/fs_project.yml:ro" \
  -v "$PWD/sources:/workspace/sources:ro" \
  -v "$PWD/transforms:/workspace/transforms:ro" \
  -v "$PWD/assets:/workspace/assets:ro" \
  -v "$PWD/policies:/workspace/policies:ro" \
  -v "$PWD/evals:/workspace/evals:ro" \
  -v "$PWD/target:/workspace/target:rw" \
  -v "$PWD/.fbt:/workspace/.fbt:rw" \
  -w /workspace \
  ghcr.io/your-org/fbt-runner-image:latest \
  fbt build --project-dir /workspace --select tag:daily
```

Use `--network=none` only for runners that do not need provider or agent
network access. OpenAI, Claude Code, Codex, Gemini, and other remote provider
runners need explicit egress if they call the provider.

## Profile 4: Network-Denied Local Run

Use for command transforms, deterministic demo runners, static document
converters, or evidence-quality scripts that should not call external services.

Controls:

- deny network at the container, VM, or host firewall layer
- omit provider credentials entirely
- set fbt policy `network: false`
- expect provider adapters to fail closed when network is unavailable or denied

fbt policy records the intended network boundary. The OS or CI environment
enforces the actual network boundary.

## Profile 5: Linux Namespace And Seccomp Wrapper

Use when Docker is too heavy but Linux process isolation is still required.

Common implementation choices include a maintained namespace wrapper, a
container runtime, or a service manager profile that can restrict filesystem
mounts, network, capabilities, and seccomp. Keep the same shape as the
container profile:

- read-only mounts for source and instruction paths
- writable mounts for `target/` and `.fbt/`
- no ambient secrets
- explicit network egress only for selected provider runners
- no write access to unrelated workspace directories

The exact command should live in the user's CI, Makefile, or internal runner
wrapper, not in fbt core.

## Profile 6: macOS Local Isolation

Use for local high-sensitivity experiments on macOS only when your organization
has a maintained sandbox wrapper for the target macOS version.

Controls should mirror the Linux/container profiles:

- allow read access to source, transform, asset, policy, and eval directories
- allow write access to `target/` and `.fbt/`
- deny writes to source directories
- deny or explicitly allow network per selected runner
- run provider or CLI-agent credentials from the shell environment only

Do not assume every macOS sandbox mechanism is stable across versions. Treat the
macOS wrapper as local execution infrastructure, not an fbt feature.

## Adapter Guidance

CLI-agent adapters such as Codex CLI and Claude Code already fail closed for
policy they cannot represent safely. That is not a substitute for OS isolation.
For high-security use, combine:

```text
fbt policy + adapter fail-closed mapping + OS sandbox profile
```

Provider SDKs and agent runtimes stay outside core. If an adapter needs special
filesystem, network, browser, or tool restrictions, implement those in the
adapter package or the launch environment and keep fbt core provider-free.

## Minimum Checklist

Before using fbt in a high-security workflow:

- choose the runner command intentionally
- run `fbt doctor`
- use read-only source mounts where possible
- keep `target/` and `.fbt/` writable together
- deny network unless the selected runner requires it
- pass only required credential environment variables
- inspect `fbt artifact explain` and failed-run receipts
- export OpenLineage or OTel only to approved destinations

