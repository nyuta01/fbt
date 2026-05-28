#!/usr/bin/env python3
"""Validate fbt plan/failure-log drift invariants."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

errors: list[str] = []


def fail(message: str) -> None:
    errors.append(message)


def exists(relative: str) -> bool:
    return (ROOT / relative).exists()


def read_text(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


def strip_fenced_code(text: str) -> str:
    return re.sub(r"```[\s\S]*?```", "", text)


def validate_plan_pdca_sections(relative_plan_path: str) -> None:
    text = read_text(relative_plan_path)
    for heading in (
        "## Observation",
        "## Decision",
        "## Permanent Fix",
        "## Next Check",
    ):
        if heading not in text:
            fail(f"{relative_plan_path}: missing required PDCA section {heading}")


def validate_quality_score(tasks: list[dict]) -> None:
    if not exists("docs/QUALITY_SCORE.md"):
        return

    task_ids = {task.get("id") for task in tasks if task.get("id")}
    text = read_text("docs/QUALITY_SCORE.md")
    rows = [line for line in text.split("\n") if line.startswith("|") and "---" not in line]

    domain_rows = 0
    for row in rows:
        cells = [cell.strip() for cell in row.split("|")[1:-1]]
        if len(cells) < 5 or cells[0] == "Domain":
            continue
        domain_rows += 1
        domain, score_text, _evidence, _weak_spot, next_task = cells[:5]
        try:
            score = int(score_text)
        except ValueError:
            fail(f"QUALITY_SCORE.md: {domain} has invalid score {score_text!r}")
            continue
        if score < 1 or score > 5:
            fail(f"QUALITY_SCORE.md: {domain} score {score} must be in 1..5")
        if score <= 2:
            match = re.search(r"`([^`]+)`", next_task)
            if not match:
                fail(f"QUALITY_SCORE.md: {domain} score {score} must name a next task")
            elif match.group(1) not in task_ids:
                fail(f"QUALITY_SCORE.md: {domain} references unknown next task {match.group(1)}")

    if domain_rows == 0:
        fail("QUALITY_SCORE.md must contain at least one scored domain row")


def validate_failure_log(tasks: list[dict]) -> None:
    if not exists("docs/agent-failures.md"):
        return

    task_ids = {task.get("id") for task in tasks if task.get("id")}
    failure_log = strip_fenced_code(read_text("docs/agent-failures.md"))
    failure_entries = re.split(r"^## ", failure_log, flags=re.MULTILINE)[1:]
    allowed_statuses = {
        "observing",
        "needs-fix",
        "fixed",
        "archived",
        "fixed-but-regressing",
    }

    for entry in failure_entries:
        heading_line, _, body = entry.partition("\n")
        id_match = re.match(r"^(F-\d{3})\b", heading_line)
        if not id_match:
            continue
        failure_id = id_match.group(1)

        status_match = re.search(r"- \*\*Status\*\*: `?([a-z-]+)`?", body)
        if not status_match:
            fail(f"{failure_id}: missing status")
            continue
        status = status_match.group(1)
        if status not in allowed_statuses:
            fail(f"{failure_id}: invalid failure status {status}")

        task_match = re.search(r"- \*\*Task\*\*: `([^`]+)`", body)
        if not task_match:
            fail(f"{failure_id}: missing linked task")
        elif task_match.group(1) not in task_ids:
            fail(f"{failure_id}: references unknown task {task_match.group(1)}")

        plan_match = re.search(r"- \*\*Plan\*\*: `([^`]+)`", body)
        if not plan_match:
            fail(f"{failure_id}: missing linked plan")
        elif not exists(plan_match.group(1)):
            fail(f"{failure_id}: references missing plan {plan_match.group(1)}")

        if status == "needs-fix":
            fail(f"{failure_id}: status is needs-fix; resolve the permanent-fix task")
        if status == "fixed" and "### Permanent fix" not in body and "**Permanent fix**" not in body:
            fail(f"{failure_id}: fixed failures must describe the permanent fix")


try:
    feature_list = json.loads(read_text("docs/exec-plans/feature-list.json"))
except FileNotFoundError:
    fail("cannot read feature-list.json: file is missing")
    feature_list = {}
except json.JSONDecodeError as exc:
    fail(f"cannot read feature-list.json: {exc}")
    feature_list = {}

if isinstance(feature_list.get("tasks"), list):
    tasks = feature_list["tasks"]
    task_ids = {task.get("id") for task in tasks if task.get("id")}
    plan_urls = {
        task.get("plan_url")
        for task in tasks
        if isinstance(task.get("plan_url"), str)
    }

    for task in tasks:
        for dep in task.get("depends", []) or []:
            if dep not in task_ids:
                fail(f"{task.get('id')}: dependency references unknown task {dep}")
        if task.get("status") in ("done", "in_progress") and not task.get("plan_url"):
            fail(f"{task.get('id')}: active or completed task must have a plan_url")
        if task.get("status") == "done" and not (task.get("paths") or []):
            fail(f"{task.get('id')}: completed task must list affected paths")

    active_plans_dir = ROOT / "docs" / "exec-plans" / "active"
    if active_plans_dir.exists():
        for entry in sorted(active_plans_dir.iterdir()):
            if entry.name == "README.md" or entry.suffix != ".md":
                continue
            relative_plan_path = f"docs/exec-plans/active/{entry.name}"
            if relative_plan_path not in plan_urls:
                fail(f"{relative_plan_path}: active plan is not referenced by feature-list")
            validate_plan_pdca_sections(relative_plan_path)

    has_permanent_fix_task = any(
        "permanent-fix" in f"{task.get('id', '')} {task.get('title', '')}".lower()
        for task in tasks
    )
    has_self_pdca_task = any(
        "pdca" in f"{task.get('id', '')} {task.get('title', '')}".lower()
        for task in tasks
    )
    if not has_permanent_fix_task:
        fail("feature-list must track the permanent-fix loop")
    if not has_self_pdca_task:
        fail("feature-list must track the self-PDCA loop")

    validate_quality_score(tasks)
    validate_failure_log(tasks)

if errors:
    print("drift-check: errors found", file=sys.stderr)
    for error in errors:
        print(f"  {error}", file=sys.stderr)
    sys.exit(1)

print("drift-check: ok")

