# fbt Release And Install Notes

Status: MVP-ready
Updated: 2026-05-30
Audience: users and maintainers installing fbt core plus optional adapters

## Core CLI

The current core CLI release is:

```text
v0.2.1
https://github.com/nyuta01/fbt/releases/tag/v0.2.1
```

Release assets include darwin, linux, and windows CLI archives for amd64 and
arm64, plus `SHA256SUMS` and `version.json`.

Install flow:

```sh
# Download the archive for your OS/arch plus SHA256SUMS from the release page.
shasum -a 256 -c SHA256SUMS
tar -xzf fbt_0.2.1_darwin_arm64.tar.gz
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

## Core Release Workflow

Core CLI releases are tag-driven. Maintainers do not manually upload archives
as the primary path.

Prepare a release candidate commit:

```sh
make release-version-check VERSION=X.Y.Z
make release-preflight RELEASE_TAG=vX.Y.Z
git tag -s vX.Y.Z -m "fbt vX.Y.Z"
git push origin main
git push origin vX.Y.Z
```

Pushing a root `vX.Y.Z` tag starts `.github/workflows/release-core.yml`. That
workflow runs the same preflight with the existing tag, runs `make verify`,
builds release archives through `scripts/release-core-cli.sh`, verifies
`SHA256SUMS`, and publishes the GitHub Release with generated notes. The
general `verify` workflow runs on pull requests and `main`; tag verification is
owned by the release workflow to avoid duplicate tag CI.

The release workflow is intentionally boring:

- signed annotated tag created by the maintainer
- repository checkout at that exact tag
- full `make verify`
- deterministic darwin/linux/windows archives
- `SHA256SUMS` and `version.json`
- `gh release create --verify-tag --generate-notes --fail-on-no-commits`

Manual fallback, for maintainers only:

```sh
scripts/release-preflight.sh --allow-existing-tag vX.Y.Z
gh release create vX.Y.Z dist/release/vX.Y.Z/* \
  --repo nyuta01/fbt \
  --title "fbt vX.Y.Z" \
  --verify-tag \
  --generate-notes \
  --latest \
  --fail-on-no-commits
```

Artifact attestations are not part of the required release baseline. If fbt
adds them later, they should be additive to the signed tag and checksum flow,
not a replacement for it.

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

Official module-scoped release tags:

| Module | Tag example | Install command |
|---|---|---|
| Go runner SDK | `sdk/go/v0.1.0` | used by adapter modules |
| Command adapter | `adapters/command/v0.1.0` | `go install github.com/nyuta01/fbt/adapters/command/cmd/fbt-runner-command@v0.1.0` |
| OpenAI adapter | `adapters/openai/v0.1.0` | `go install github.com/nyuta01/fbt/adapters/openai/cmd/fbt-runner-openai@v0.1.0` |
| Codex CLI adapter | `adapters/codex-cli/v0.1.0` | `go install github.com/nyuta01/fbt/adapters/codex-cli/cmd/fbt-runner-codex-cli@v0.1.0` |
| Claude Code adapter | `adapters/claude-code/v0.1.0` | `go install github.com/nyuta01/fbt/adapters/claude-code/cmd/fbt-runner-claude-code@v0.1.0` |

The root `v0.2.1` tag belongs to fbt core CLI releases. Do not use root tags as
adapter module versions.

## Adapter Release Workflow

Adapters currently ship as Go module source installs. The release integrity
baseline is:

1. Run source checks.
2. Create a module-scoped signed annotated tag.
3. Verify the tag signature.
4. Verify the module checksum through Go's module machinery.
5. Publish binary checksums/signatures only if a future adapter release adds
   binary archives.

Preflight:

```sh
make verify
make official-adapter-smoke
make adapter-install-smoke
make adapter-release-plan-check
```

Cut and verify a module-scoped signed tag:

```sh
git tag -s adapters/openai/v0.1.0 -m "adapters/openai v0.1.0"
git tag -v adapters/openai/v0.1.0
git push origin adapters/openai/v0.1.0
```

Verify the install and module checksum:

```sh
go install github.com/nyuta01/fbt/adapters/openai/cmd/fbt-runner-openai@v0.1.0
go env GOSUMDB
go mod download -json github.com/nyuta01/fbt/adapters/openai@v0.1.0
```

The Go checksum database and the user's `go.sum` entry cover source module
integrity for `go install` users. If an adapter later publishes binary
archives, the release must include adapter-specific `SHA256SUMS` and a detached
signature. Recommended signing command:

```sh
shasum -a 256 fbt-runner-openai_* > SHA256SUMS
cosign sign-blob --output-signature SHA256SUMS.sig SHA256SUMS
cosign verify-blob --signature SHA256SUMS.sig SHA256SUMS
```

Keep adapter `SHA256SUMS` separate from the core CLI `SHA256SUMS` so users can
verify only the package they install.

Each adapter release note should state:

- module path and module-scoped tag
- fbt core version tested with the adapter
- runner protocol version
- executable command name
- logical runner names
- required credential environment variables
- provider API or CLI-agent version tested
- verification commands and results

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
