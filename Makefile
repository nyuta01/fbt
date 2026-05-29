# Lightweight harness entrypoint for fbt.
#
# Keep this file as the single command surface for agents. Product-specific
# checks should be added behind `make verify` as implementation lands.

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS += --no-print-directory

GO ?= go
GOFMT ?= gofmt
PYTHON ?= python3
VERSION ?= 0.2.1
RELEASE_TAG ?= v$(VERSION)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
FBT_LDFLAGS := -X github.com/nyuta01/fbt/internal/version.Version=$(VERSION) -X github.com/nyuta01/fbt/internal/version.Commit=$(COMMIT) -X github.com/nyuta01/fbt/internal/version.BuildDate=$(BUILD_DATE)
GOFMT_DIRS := cmd internal examples/runner_adapters tests/runner_fixtures sdk/go adapters/command adapters/openai adapters/codex-cli adapters/claude-code

.DEFAULT_GOAL := help

.PHONY: help
help: ## List documented targets.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: agent-init
agent-init: ## Print restart context and run the current verification gate.
	@bash scripts/agent-init.sh

.PHONY: harness-check
harness-check: ## Validate repository harness shape and structured task state.
	@$(PYTHON) scripts/harness_check.py

.PHONY: drift-check
drift-check: ## Validate plan/failure-log drift invariants.
	@$(PYTHON) scripts/harness_drift.py

.PHONY: validate-docs
validate-docs: ## Validate docs-local links and language/file-name invariants.
	@$(PYTHON) scripts/validate_docs.py

.PHONY: project-config-schema-check
project-config-schema-check: ## Verify generated project-config JSON Schemas.
	@$(PYTHON) scripts/generate-project-config-schema.py --check

.PHONY: adapter-release-plan-check
adapter-release-plan-check: ## Verify official adapter release docs and workflow.
	@$(PYTHON) scripts/check-adapter-release-plan.py

.PHONY: security-profiles-check
security-profiles-check: ## Verify external OS sandbox profile docs.
	@$(PYTHON) scripts/check-security-profiles.py

.PHONY: runner-lockfile-spec-check
runner-lockfile-spec-check: ## Verify optional runner lockfile spec boundaries.
	@$(PYTHON) scripts/check-runner-lockfile-spec.py

.PHONY: release-version-check
release-version-check: ## Verify core release version references and release workflow shape.
	@$(PYTHON) scripts/check-release-version.py "$(VERSION)"

.PHONY: agent-skills-check
agent-skills-check: ## Verify distributable skills.sh-compatible agent skills.
	@$(PYTHON) scripts/check-agent-skills.py

.PHONY: release-preflight
release-preflight: ## Verify and build release assets for RELEASE_TAG=vX.Y.Z.
	@bash scripts/release-preflight.sh "$(RELEASE_TAG)"

.PHONY: fmt
fmt: ## Format Go source.
	@$(GOFMT) -w $(GOFMT_DIRS)

.PHONY: fmt-check
fmt-check: ## Verify Go source formatting.
	@test -z "$$($(GOFMT) -l $(GOFMT_DIRS))"

.PHONY: go-test
go-test: ## Run Go unit tests.
	@$(GO) test ./...

.PHONY: sdk-go-test
sdk-go-test: ## Run provider-free Go runner SDK tests.
	@cd sdk/go && $(GO) test ./...

.PHONY: adapter-command-test
adapter-command-test: ## Run official command adapter tests.
	@cd adapters/command && $(GO) test ./...

.PHONY: adapter-command-conformance
adapter-command-conformance: ## Run conformance against the official command adapter.
	@FBT_COMMAND_ADAPTER_DEFAULT_COMMAND="$(CURDIR)/adapters/command/testdata/write-output.sh" $(PYTHON) tests/runner-conformance/run.py --runner-command 'go run ./adapters/command/cmd/fbt-runner-command' --transform-type command --strict

.PHONY: adapter-openai-test
adapter-openai-test: ## Run official OpenAI adapter tests.
	@cd adapters/openai && $(GO) test ./...

.PHONY: adapter-openai-conformance
adapter-openai-conformance: ## Run conformance against the official OpenAI adapter without a provider call.
	@OPENAI_API_KEY=test FBT_OPENAI_ADAPTER_FAKE_RESPONSE="# OpenAI Adapter Conformance" $(PYTHON) tests/runner-conformance/run.py --runner-command 'go run ./adapters/openai/cmd/fbt-runner-openai' --transform-type llm --strict

.PHONY: adapter-codex-cli-test
adapter-codex-cli-test: ## Run official Codex CLI adapter tests.
	@cd adapters/codex-cli && $(GO) test ./...

.PHONY: adapter-codex-cli-conformance
adapter-codex-cli-conformance: ## Run agent conformance against the official Codex CLI adapter with a fixture CLI.
	@FBT_CODEX_CLI_COMMAND="$(CURDIR)/adapters/codex-cli/testdata/codex-cli-fixture.sh" $(PYTHON) tests/runner-conformance/run.py --runner-command 'go run ./adapters/codex-cli/cmd/fbt-runner-codex-cli' --transform-type agent --strict --agent-adapter
	@FBT_CODEX_CLI_COMMAND="$(CURDIR)/adapters/codex-cli/testdata/codex-cli-fixture.sh" $(PYTHON) tests/runner-conformance/run.py --runner-command 'go run ./adapters/codex-cli/cmd/fbt-runner-codex-cli' --transform-type agent --strict --agent-adapter --expect-policy-failure

.PHONY: adapter-claude-code-test
adapter-claude-code-test: ## Run official Claude Code adapter tests.
	@cd adapters/claude-code && $(GO) test ./...

.PHONY: adapter-claude-code-conformance
adapter-claude-code-conformance: ## Run agent conformance against the official Claude Code adapter with a fixture CLI.
	@FBT_CLAUDE_CODE_COMMAND="$(CURDIR)/adapters/claude-code/testdata/claude-code-fixture.sh" $(PYTHON) tests/runner-conformance/run.py --runner-command 'go run ./adapters/claude-code/cmd/fbt-runner-claude-code' --transform-type agent --strict --agent-adapter
	@FBT_CLAUDE_CODE_COMMAND="$(CURDIR)/adapters/claude-code/testdata/claude-code-fixture.sh" $(PYTHON) tests/runner-conformance/run.py --runner-command 'go run ./adapters/claude-code/cmd/fbt-runner-claude-code' --transform-type agent --strict --agent-adapter --expect-policy-failure

.PHONY: official-adapter-smoke
official-adapter-smoke: adapter-command-conformance adapter-openai-conformance adapter-codex-cli-conformance adapter-claude-code-conformance ## Verify official adapters with fixtures/fake responses and no live provider calls.
	@echo "official-adapter-smoke: ok"

.PHONY: build
build: ## Build the fbt CLI into bin/fbt.
	@mkdir -p bin
	@$(GO) build -ldflags "$(FBT_LDFLAGS)" -o bin/fbt ./cmd/fbt

.PHONY: cli-smoke
cli-smoke: ## Run a deterministic fbt CLI smoke.
	@bash scripts/smoke-cli.sh

.PHONY: install-script-smoke
install-script-smoke: ## Verify the release installer against a local archive.
	@bash scripts/smoke-install-script.sh

.PHONY: e2e-smoke
e2e-smoke: ## Run the local knowledge-loop smoke.
	@bash scripts/smoke-knowledge-loop.sh

.PHONY: practical-examples-smoke
practical-examples-smoke: ## Plan external-runner practical examples without provider calls.
	@bash scripts/smoke-practical-examples.sh

.PHONY: own-files-smoke
own-files-smoke: ## Verify the first own-files user path.
	@bash scripts/smoke-own-files.sh

.PHONY: daily-ops-smoke
daily-ops-smoke: ## Verify daily growing-source operations with multiple artifacts.
	@bash scripts/smoke-daily-ops.sh

.PHONY: semantic-eval-boundary-smoke
semantic-eval-boundary-smoke: ## Verify external semantic/evidence quality checks stay outside core.
	@bash scripts/smoke-semantic-eval-boundary.sh

.PHONY: retention-high-volume-smoke
retention-high-volume-smoke: ## Verify retention inspection under many artifact versions.
	@bash scripts/smoke-retention-high-volume.sh

.PHONY: real-llm-smoke
real-llm-smoke: build ## Run opt-in smoke against an external real LLM runner.
	@FBT_BIN="$(CURDIR)/bin/fbt" bash scripts/smoke-real-llm.sh

.PHONY: runner-adapter-smoke
runner-adapter-smoke: build ## Run opt-in smoke matrix against installed runner adapters.
	@FBT_BIN="$(CURDIR)/bin/fbt" bash scripts/smoke-runner-adapters.sh

.PHONY: adapter-install-smoke
adapter-install-smoke: ## Verify official adapter commands install from a clean VCS module fetch.
	@bash scripts/smoke-adapter-install.sh

.PHONY: standard-backend-smoke
standard-backend-smoke: build ## Run opt-in smoke against standard visualization backends.
	@FBT_BIN="$(CURDIR)/bin/fbt" bash scripts/smoke-standard-backends.sh

.PHONY: docs-site-build
docs-site-build: ## Build the Astro/Starlight documentation site.
	@cd apps/docs && npm ci --no-audit --fund=false && SITE="https://nyuta01.github.io" BASE="/fbt" npm run build

.PHONY: runner-conformance
runner-conformance: ## Run the minimal external runner protocol conformance fixture.
	@bash scripts/runner-conformance.sh --strict

.PHONY: runner-scaffold-conformance
runner-scaffold-conformance: ## Run conformance against the copyable runner scaffold.
	@$(PYTHON) tests/runner-conformance/run.py --runner-command examples/runner_adapter_scaffold/bin/fbt-runner-example --strict --agent-adapter

.PHONY: conformance
conformance: build ## Run structured deterministic MVP conformance scenarios.
	@FBT_BIN="$(CURDIR)/bin/fbt" $(PYTHON) tests/conformance/run.py

.PHONY: dist-check
dist-check: ## Build and smoke the local release binary.
	@VERSION="$(VERSION)" COMMIT="$(COMMIT)" BUILD_DATE="$(BUILD_DATE)" bash scripts/dist-check.sh

.PHONY: verify
verify: harness-check drift-check validate-docs project-config-schema-check adapter-release-plan-check security-profiles-check runner-lockfile-spec-check release-version-check agent-skills-check fmt-check go-test sdk-go-test adapter-command-test adapter-command-conformance adapter-openai-test adapter-openai-conformance adapter-codex-cli-test adapter-codex-cli-conformance adapter-claude-code-test adapter-claude-code-conformance cli-smoke install-script-smoke e2e-smoke practical-examples-smoke own-files-smoke daily-ops-smoke semantic-eval-boundary-smoke retention-high-volume-smoke docs-site-build runner-conformance runner-scaffold-conformance conformance dist-check ## Run the current single verification gate.
	@echo "verify: ok"
