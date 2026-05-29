#!/usr/bin/env python3
"""Run production-shaped quality gates over a daily_qa_ops run bundle."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


def gate(name: str, status: str, evidence: str, required: bool = True) -> dict:
    return {
        "name": name,
        "status": status,
        "required": required,
        "evidence": evidence,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--project-dir", required=True)
    parser.add_argument("--run-dir", required=True)
    args = parser.parse_args()

    project_dir = Path(args.project_dir).resolve()
    run_dir = Path(args.run_dir).resolve()
    artifacts = [
        project_dir / "target/artifacts/qa/latest/faq_candidates.md",
        project_dir / "target/artifacts/qa/latest/manual_patch_candidates.md",
        project_dir / "target/artifacts/qa/latest/unresolved_questions.md",
        project_dir / "target/artifacts/manual/latest/manual_update.md",
    ]

    results: list[dict] = []
    missing = [str(path.relative_to(project_dir)) for path in artifacts if not path.is_file()]
    empty = [
        str(path.relative_to(project_dir))
        for path in artifacts
        if path.is_file() and path.stat().st_size == 0
    ]
    if missing or empty:
        results.append(gate("structural_artifacts", "fail", f"missing={missing} empty={empty}"))
    else:
        results.append(gate("structural_artifacts", "pass", "all declared artifacts exist and are non-empty"))

    explain_path = run_dir / "manual_update-explain.txt"
    openlineage_path = run_dir / "openlineage.ndjson"
    explain = explain_path.read_text(encoding="utf-8") if explain_path.exists() else ""
    openlineage = openlineage_path.read_text(encoding="utf-8") if openlineage_path.exists() else ""
    required_markers = [
        "manual_patch_candidates",
        "unresolved_questions",
        "reference.current_manual",
        "data/qa/inbox/questions",
        "data/qa/inbox/answers",
    ]
    missing_markers = [
        marker
        for marker in required_markers
        if marker not in explain and marker not in openlineage
    ]
    if missing_markers:
        results.append(gate("evidence_lineage", "fail", f"missing lineage markers={missing_markers}"))
    else:
        results.append(gate("evidence_lineage", "pass", "artifact explain and OpenLineage expose upstream evidence"))

    results.append(
        gate(
            "domain_review",
            "pending",
            "manual_update requires external owner review before publishing",
            required=False,
        )
    )

    status = "pass" if all(item["status"] != "fail" for item in results) else "fail"
    payload = {"status": status, "gates": results}
    (run_dir / "quality-gates.json").write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")

    print("Quality Gates")
    for item in results:
        print(f"  {item['status'].upper():7} {item['name']} - {item['evidence']}")
    print(f"  RESULT  {status}")
    return 0 if status == "pass" else 5


if __name__ == "__main__":
    raise SystemExit(main())
