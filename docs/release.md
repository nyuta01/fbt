# fbt Release And Install Notes

Status: MVP-ready
Updated: 2026-05-29
Audience: users and maintainers installing fbt core plus optional adapters

## Core CLI

The current MVP release is:

```text
v0.1.0
https://github.com/nyuta01/fbt/releases/tag/v0.1.0
```

Release assets include darwin, linux, and windows CLI archives for amd64 and
arm64, plus `SHA256SUMS`.

Install flow:

```sh
# Download the archive for your OS/arch plus SHA256SUMS from the release page.
shasum -a 256 -c SHA256SUMS
tar -xzf fbt_0.1.0_darwin_arm64.tar.gz
install -m 0755 fbt "$HOME/.local/bin/fbt"
fbt version
```

Use the matching archive name for Linux, Windows, amd64, or arm64.

Source build:

```sh
git clone https://github.com/nyuta01/fbt.git
cd fbt
make build
./bin/fbt version
```

## Official Adapters

fbt core is provider-free. Install runner adapters separately:

```sh
go install github.com/nyuta01/fbt/adapters/command/cmd/fbt-runner-command@main
go install github.com/nyuta01/fbt/adapters/openai/cmd/fbt-runner-openai@main
go install github.com/nyuta01/fbt/adapters/codex-cli/cmd/fbt-runner-codex-cli@main
go install github.com/nyuta01/fbt/adapters/claude-code/cmd/fbt-runner-claude-code@main
```

For production, pin a commit or an adapter module tag when one is published.
Adapter tags should be scoped to the nested module, for example
`adapters/openai/v0.1.0`.

## Verification

Source checkout checks:

```sh
make verify
make official-adapter-smoke
make adapter-install-smoke
```

Installed adapter check without a live build:

```sh
export OPENAI_API_KEY=...
FBT_RUNNER_ADAPTER_SMOKE_MATRIX='openai.responses|llm|markdown|fbt-runner-openai responses|OPENAI_API_KEY|false' \
make runner-adapter-smoke
```

Live adapter build check:

```sh
FBT_RUNNER_ADAPTER_SMOKE_BUILD=1 \
FBT_RUNNER_ADAPTER_SMOKE_MATRIX='openai.responses|llm|markdown|fbt-runner-openai responses|OPENAI_API_KEY|false' \
make runner-adapter-smoke
```

Only use the live build check when you intend to call the real provider or
agent and create a temporary artifact.

## Compatibility Contract

The compatibility boundary is the fbt runner protocol. A release should state:

- fbt core version
- runner protocol version
- adapter module path and version or commit
- logical runner names provided by each adapter
- required credential environment variable names
- provider API or agent CLI version tested by the adapter

Core release artifacts and adapter commands can be distributed independently as
long as the runner protocol remains compatible.
