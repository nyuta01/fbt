#!/usr/bin/env python3
"""Check that the runner production reliability contract stays wired."""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

REQUIRED_TEXT = {
    "docs/runner-production-reliability.md": [
        "Input-size failure",
        "Retry classification",
        "Provider metadata",
        "Redaction",
        "Timeout mapping",
        "Fail-closed policy",
        "Output boundary",
        "Live opt-in",
        "make production-pilot-smoke",
    ],
    "docs/runner-protocol-spec.md": [
        "Production Reliability",
        "retryable",
        "fail closed",
        "token usage",
        "cost",
    ],
    "docs/runner-adapters.md": [
        "Production Reliability Baseline",
        "fbt-runner-openai",
        "fbt-runner-codex-cli",
        "fbt-runner-claude-code",
    ],
    "tests/runner-conformance/README.md": [
        "Production Reliability",
        "oversized source",
        "policy failure",
        "redaction marker",
    ],
    "Makefile": [
        "runner-production-reliability-check",
        "production-pilot-smoke",
    ],
}

CODE_MARKERS = {
    "adapters/codex-cli/internal/codexcliadapter/adapter.go": [
        "stagedInputMaxBytes",
        "policy_fail_closed",
        "Timeout",
        "cannot enforce",
    ],
    "adapters/claude-code/internal/claudecodeadapter/adapter.go": [
        "stagedInputMaxBytes",
        "policy_fail_closed",
        "Timeout",
        "cannot enforce",
    ],
    "adapters/openai/internal/openaiadapter/adapter.go": [
        "OPENAI_API_KEY",
        "FBT_OPENAI_ADAPTER_FAKE_RESPONSE",
        "usage",
        "fbt.usage.total_tokens",
    ],
}


def main() -> int:
    errors: list[str] = []
    for relative, needles in REQUIRED_TEXT.items():
        text = (ROOT / relative).read_text(encoding="utf-8")
        for needle in needles:
            if needle not in text:
                errors.append(f"{relative}: missing {needle!r}")
    for relative, needles in CODE_MARKERS.items():
        text = (ROOT / relative).read_text(encoding="utf-8")
        for needle in needles:
            if needle not in text:
                errors.append(f"{relative}: missing code marker {needle!r}")
    if errors:
        print("runner-production-reliability-check: errors found", file=sys.stderr)
        for error in errors:
            print(f"  {error}", file=sys.stderr)
        return 1
    print("runner-production-reliability-check: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
