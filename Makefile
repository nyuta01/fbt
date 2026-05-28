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
	@$(GOFMT) -w cmd internal runners

.PHONY: fmt-check
fmt-check: ## Verify Go source formatting.
	@test -z "$$($(GOFMT) -l cmd internal runners)"

.PHONY: go-test
go-test: ## Run Go unit tests.
	@$(GO) test ./...

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

.PHONY: real-llm-smoke
real-llm-smoke: build ## Run opt-in smoke against an external real LLM runner.
	@FBT_BIN="$(CURDIR)/bin/fbt" bash scripts/smoke-real-llm.sh

.PHONY: conformance
conformance: build ## Run deterministic MVP conformance scenarios.
	@FBT_BIN="$(CURDIR)/bin/fbt" bash tests/conformance/run.sh

.PHONY: dist-check
dist-check: ## Build and smoke the local release binary.
	@VERSION="$(VERSION)" COMMIT="$(COMMIT)" BUILD_DATE="$(BUILD_DATE)" bash scripts/dist-check.sh

.PHONY: verify
verify: harness-check drift-check validate-docs fmt-check go-test cli-smoke e2e-smoke conformance dist-check ## Run the current single verification gate.
	@echo "verify: ok"
