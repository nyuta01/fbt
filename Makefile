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
VERSION ?= 0.1.0
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

.PHONY: adapter-claude-code-test
adapter-claude-code-test: ## Run official Claude Code adapter tests.
	@cd adapters/claude-code && $(GO) test ./...

.PHONY: adapter-claude-code-conformance
adapter-claude-code-conformance: ## Run agent conformance against the official Claude Code adapter with a fixture CLI.
	@FBT_CLAUDE_CODE_COMMAND="$(CURDIR)/adapters/claude-code/testdata/claude-code-fixture.sh" $(PYTHON) tests/runner-conformance/run.py --runner-command 'go run ./adapters/claude-code/cmd/fbt-runner-claude-code' --transform-type agent --strict --agent-adapter

.PHONY: build
build: ## Build the fbt CLI into bin/fbt.
	@mkdir -p bin
	@$(GO) build -ldflags "$(FBT_LDFLAGS)" -o bin/fbt ./cmd/fbt

.PHONY: cli-smoke
cli-smoke: ## Run a deterministic fbt CLI smoke.
	@bash scripts/smoke-cli.sh

.PHONY: e2e-smoke
e2e-smoke: ## Run the local knowledge-loop smoke.
	@bash scripts/smoke-knowledge-loop.sh

.PHONY: practical-examples-smoke
practical-examples-smoke: ## Plan external-runner practical examples without provider calls.
	@bash scripts/smoke-practical-examples.sh

.PHONY: real-llm-smoke
real-llm-smoke: build ## Run opt-in smoke against an external real LLM runner.
	@FBT_BIN="$(CURDIR)/bin/fbt" bash scripts/smoke-real-llm.sh

.PHONY: runner-adapter-smoke
runner-adapter-smoke: build ## Run opt-in smoke matrix against installed runner adapters.
	@FBT_BIN="$(CURDIR)/bin/fbt" bash scripts/smoke-runner-adapters.sh

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
conformance: build ## Run deterministic MVP conformance scenarios.
	@FBT_BIN="$(CURDIR)/bin/fbt" bash tests/conformance/run.sh

.PHONY: dist-check
dist-check: ## Build and smoke the local release binary.
	@VERSION="$(VERSION)" COMMIT="$(COMMIT)" BUILD_DATE="$(BUILD_DATE)" bash scripts/dist-check.sh

.PHONY: verify
verify: harness-check drift-check validate-docs fmt-check go-test sdk-go-test adapter-command-test adapter-command-conformance adapter-openai-test adapter-openai-conformance adapter-codex-cli-test adapter-codex-cli-conformance adapter-claude-code-test adapter-claude-code-conformance cli-smoke e2e-smoke practical-examples-smoke docs-site-build runner-conformance runner-scaffold-conformance conformance dist-check ## Run the current single verification gate.
	@echo "verify: ok"
