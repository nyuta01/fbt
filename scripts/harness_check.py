#!/usr/bin/env python3
"""Validate fbt repository harness shape and structured task state."""

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

errors: list[str] = []


def fail(message: str) -> None:
    errors.append(message)


def exists(relative: str) -> bool:
    return (ROOT / relative).exists()


def has_tracked_content(relative: str) -> bool:
    result = subprocess.run(
        ["git", "ls-files", "--", relative],
        cwd=ROOT,
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
    )
    return bool(result.stdout.strip())


def read_text(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


REQUIRED_FILES = [
    "AGENTS.md",
    "AGENT_PROGRESS.md",
    "Makefile",
    "README.md",
    "LICENSE",
    "CONTRIBUTING.md",
    "SECURITY.md",
    "CODE_OF_CONDUCT.md",
    ".gitignore",
    ".github/workflows/verify.yml",
    "go.mod",
    "cmd/fbt/main.go",
    "internal/README.md",
    "internal/cli/cli.go",
    "internal/cli/cli_test.go",
    "docs/design-doc.md",
    "docs/spec.md",
    "docs/project-config-spec.md",
    "docs/runner-protocol-spec.md",
    "docs/schema-and-versioning-spec.md",
    "docs/runner-discovery-spec.md",
    "docs/security-and-conformance-spec.md",
    "docs/state-and-run-results-spec.md",
    "docs/methodology/harness-engineering.md",
    "docs/methodology/permanent-fix-protocol.md",
    "docs/methodology/self-pdca-loop.md",
    "docs/exec-plans/README.md",
    "docs/exec-plans/active/README.md",
    "docs/exec-plans/feature-list.json",
    "docs/exec-plans/active/FBT-H-001-plan.md",
    "docs/QUALITY_SCORE.md",
    "docs/agent-failures.md",
    "scripts/agent-init.sh",
    "scripts/harness_check.py",
    "scripts/harness_drift.py",
    "scripts/validate_docs.py",
    "scripts/smoke-cli.sh",
]

for relative in REQUIRED_FILES:
    if not exists(relative):
        fail(f"missing required harness file: {relative}")


if exists("AGENTS.md"):
    line_count = len(read_text("AGENTS.md").rstrip("\n").split("\n"))
    if line_count > 120:
        fail(f"AGENTS.md must stay compact; found {line_count} lines, max 120")


if exists("Makefile"):
    makefile = read_text("Makefile")
    for target_fragment in (
        r"^verify:.*harness-check",
        r"^verify:.*drift-check",
        r"^verify:.*validate-docs",
        r"^verify:.*fmt-check",
        r"^verify:.*go-test",
        r"^verify:.*cli-smoke",
    ):
        if not re.search(target_fragment, makefile, re.MULTILINE):
            fail(f"Makefile verify target must match {target_fragment}")


if exists(".github/workflows/verify.yml"):
    workflow = read_text(".github/workflows/verify.yml")
    if "pull_request:" not in workflow:
        fail(".github/workflows/verify.yml must run on pull_request")
    if "branches:" not in workflow:
        fail(".github/workflows/verify.yml must constrain push branches")
    for required_text in ("actions/setup-go@", "python-version:", "make verify"):
        if required_text not in workflow:
            fail(f".github/workflows/verify.yml must include {required_text}")
    for action, ref in re.findall(r"uses:\s*(actions/[^@\s]+)@([0-9A-Za-z_.-]+)", workflow):
        if not re.fullmatch(r"[0-9a-f]{40}", ref):
            fail(f".github/workflows/verify.yml must pin {action} by full commit SHA")


feature_list: dict | None = None
if exists("docs/exec-plans/feature-list.json"):
    try:
        feature_list = json.loads(read_text("docs/exec-plans/feature-list.json"))
    except json.JSONDecodeError as exc:
        fail(f"feature-list.json is not valid JSON: {exc}")


def validate_tasks(tasks: list[dict]) -> None:
    seen_ids: set[str] = set()
    allowed_statuses = {"todo", "in_progress", "blocked", "done"}
    allowed_priorities = {"P0", "P1", "P2"}

    for task in tasks:
        prefix = task.get("id", "<missing-id>")
        for field in ("id", "title", "priority", "status", "owner"):
            value = task.get(field)
            if not isinstance(value, str) or not value:
                fail(f"{prefix}: missing string field {field}")

        task_id = task.get("id")
        if isinstance(task_id, str):
            if task_id in seen_ids:
                fail(f"{prefix}: duplicate task id")
            seen_ids.add(task_id)

        if task.get("priority") not in allowed_priorities:
            fail(f"{prefix}: invalid priority {task.get('priority')!r}")
        if task.get("status") not in allowed_statuses:
            fail(f"{prefix}: invalid status {task.get('status')!r}")
        if not isinstance(task.get("depends"), list):
            fail(f"{prefix}: depends must be an array")

        plan_url = task.get("plan_url")
        if plan_url is not None and not isinstance(plan_url, str):
            fail(f"{prefix}: plan_url must be a string or null")
        if isinstance(plan_url, str) and not exists(plan_url):
            fail(f"{prefix}: plan_url references missing file {plan_url}")

        paths = task.get("paths")
        if not isinstance(paths, list):
            fail(f"{prefix}: paths must be an array")
        elif task.get("status") == "done":
            for task_path in paths:
                if not exists(task_path):
                    fail(f"{prefix}: done task references missing path {task_path}")
                elif not has_tracked_content(task_path):
                    fail(f"{prefix}: done task references untracked path {task_path}")

        verification = task.get("verification")
        if not isinstance(verification, dict):
            fail(f"{prefix}: verification must be an object")
        elif task.get("status") == "done":
            for gate, value in verification.items():
                if value is False and not task.get("verification_override_reason"):
                    fail(f"{prefix}: done task has false verification gate {gate}")


if feature_list is not None:
    if feature_list.get("schema_version") != 1:
        fail("feature-list.json schema_version must be 1")
    tasks = feature_list.get("tasks")
    if not isinstance(tasks, list):
        fail("feature-list.json tasks must be an array")
    else:
        validate_tasks(tasks)


if errors:
    print("harness-check: errors found", file=sys.stderr)
    for error in errors:
        print(f"  {error}", file=sys.stderr)
    sys.exit(1)

print("harness-check: ok")
