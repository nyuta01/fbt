#!/usr/bin/env python3
from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[2]
FBT_BIN = Path(os.environ.get("FBT_BIN", ROOT_DIR / "bin" / "fbt"))


@dataclass(frozen=True)
class CommandResult:
    args: list[str]
    returncode: int
    stdout: str
    stderr: str


@dataclass(frozen=True)
class Context:
    tmpdir: Path


@dataclass(frozen=True)
class Scenario:
    name: str
    run: Callable[[Context], None]


class ConformanceError(AssertionError):
    pass


def fail(message: str) -> None:
    raise ConformanceError(message)


def check(condition: bool, message: str) -> None:
    if not condition:
        fail(message)


def assert_contains(text: str, needle: str, label: str) -> None:
    if needle not in text:
        fail(f"{label} missing {needle!r}\n{text}")


def assert_regex(text: str, pattern: str, label: str) -> None:
    if not re.search(pattern, text, re.MULTILINE):
        fail(f"{label} missing pattern {pattern!r}\n{text}")


def assert_file(path: Path) -> None:
    check(path.is_file(), f"expected file to exist: {path}")


def assert_no_path(path: Path) -> None:
    check(not path.exists(), f"expected path not to exist: {path}")


def write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def append(path: Path, text: str) -> None:
    with path.open("a", encoding="utf-8") as fh:
        fh.write(text)


def run_cmd(args: list[str], *, expect: int | None = 0, env: dict[str, str] | None = None) -> CommandResult:
    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)
    result = subprocess.run(
        args,
        cwd=ROOT_DIR,
        env=merged_env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    command = " ".join(args)
    if expect is not None and result.returncode != expect:
        fail(
            f"command exited {result.returncode}, expected {expect}: {command}\n"
            f"stdout:\n{result.stdout}\n"
            f"stderr:\n{result.stderr}"
        )
    return CommandResult(args=args, returncode=result.returncode, stdout=result.stdout, stderr=result.stderr)


def fbt(args: list[str], *, expect: int | None = 0, env: dict[str, str] | None = None) -> CommandResult:
    return run_cmd([str(FBT_BIN), *args], expect=expect, env=env)


def init_project(path: Path) -> None:
    fbt(["init", str(path), "--template", "support"])


def scenario_config_version(ctx: Context) -> None:
    missing = ctx.tmpdir / "schema-missing"
    missing.mkdir()
    write(missing / "fs_project.yml", "name: schema_missing\n")
    result = fbt(["plan", "--project-dir", str(missing)], expect=2)
    assert_contains(result.stderr, "CONFIG_VERSION_MISSING", "missing config_version stderr")

    unsupported = ctx.tmpdir / "schema-unsupported"
    unsupported.mkdir()
    write(unsupported / "fs_project.yml", "name: schema_unsupported\nconfig_version: 999\n")
    result = fbt(["plan", "--project-dir", str(unsupported)], expect=2)
    assert_contains(result.stderr, "CONFIG_VERSION_UNSUPPORTED", "unsupported config_version stderr")


def expect_yaml_unknown(project: Path, label: str) -> None:
    result = fbt(["plan", "--project-dir", str(project)], expect=2)
    assert_contains(result.stderr, "YAML_FIELD_UNKNOWN", f"{label} stderr")


def scenario_strict_yaml(ctx: Context) -> None:
    unknown_project = ctx.tmpdir / "unknown-project"
    unknown_project.mkdir()
    write(
        unknown_project / "fs_project.yml",
        "name: unknown_project\nconfig_version: 1\nsorce_paths: [\"sources\"]\n",
    )
    expect_yaml_unknown(unknown_project, "unknown project field")

    unknown_runner = ctx.tmpdir / "unknown-runner"
    init_project(unknown_runner)
    write(
        unknown_runner / "fs_project.yml",
        """name: unknown_runner
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
runners:
  - name: demo.llm
    type: llm
    protocol: stdio_jsonrpc
    cmd: bin/fbt-demo-llm-runner
""",
    )
    expect_yaml_unknown(unknown_runner, "unknown runner field")

    unknown_source = ctx.tmpdir / "unknown-source"
    init_project(unknown_source)
    write(
        unknown_source / "sources" / "support.yml",
        """sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        pth: data/support/tickets/*.jsonl
""",
    )
    expect_yaml_unknown(unknown_source, "unknown source field")

    unknown_transform = ctx.tmpdir / "unknown-transform"
    init_project(unknown_transform)
    write(
        unknown_transform / "transforms" / "support" / "case_summaries.yml",
        """transforms:
  - name: case_summaries
    type: llm
    runner: demo.llm
    modle:
      provider: demo
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
""",
    )
    expect_yaml_unknown(unknown_transform, "unknown transform field")

    unknown_policy = ctx.tmpdir / "unknown-policy"
    init_project(unknown_policy)
    write(
        unknown_policy / "policies" / "support.yml",
        """policies:
  - name: support_agent_scope
    read: ["data/support/"]
    write: [".fbt/work/", "target/artifacts/support/"]
    netwrok: true
""",
    )
    expect_yaml_unknown(unknown_policy, "unknown policy field")

    unknown_eval = ctx.tmpdir / "unknown-eval"
    init_project(unknown_eval)
    write(
        unknown_eval / "evals" / "support.yml",
        """evals:
  - name: required_case_sections
    type: deterministic
    grant_confidence: structural
    config:
      sections: ["Fake Output"]
""",
    )
    expect_yaml_unknown(unknown_eval, "unknown eval field")


def scenario_build_lifecycle(ctx: Context) -> Path:
    happy = ctx.tmpdir / "happy"
    init_project(happy)

    dag = ctx.tmpdir / "dag"
    init_project(dag)
    dag_build = fbt(["build", "--project-dir", str(dag), "--select", "+weekly_support_insights"])
    assert_contains(dag_build.stdout, "run       2", "DAG build stdout")
    assert_file(dag / "target" / "artifacts" / "support" / "case_summaries" / "index.md")
    assert_file(dag / "target" / "artifacts" / "support" / "weekly_insights.md")

    redaction_marker = "FBT_CONFORMANCE_SECRET_DO_NOT_EXPORT"
    append(happy / "assets" / "support_style_guide.md", f"\n- Do not export marker: {redaction_marker}\n")
    append(
        happy / "data" / "support" / "tickets" / "2026-05-28.jsonl",
        f'{{"id":"T-secret","summary":"{redaction_marker}","impact":"redaction fixture"}}\n',
    )
    build_case = fbt(["build", "--project-dir", str(happy), "--select", "case_summaries"])
    assert_contains(build_case.stdout, "SUCCESS case_summaries", "case build stdout")
    assert_file(happy / "target" / "artifacts" / "support" / "case_summaries" / "index.md")

    show = fbt(["artifact", "show", "case_summaries", "--project-dir", str(happy)])
    assert_contains(show.stdout, "Semantic summary  ", "artifact show stdout")

    retention = fbt(["artifact", "retention", "--project-dir", str(happy)])
    assert_contains(retention.stdout, "Policy               keep_all", "retention stdout")
    assert_contains(retention.stdout, "Artifact versions    1", "retention stdout")
    assert_contains(retention.stdout, "Action               no files removed", "retention stdout")

    policy_decisions = (happy / ".fbt" / "state" / "policy_decisions.json").read_text(encoding="utf-8")
    assert_contains(policy_decisions, '"status": "allowed"', "policy decisions")

    second_build = fbt(["build", "--project-dir", str(happy), "--select", "case_summaries"])
    assert_contains(second_build.stdout, "selected  1", "second build stdout")
    assert_contains(second_build.stdout, "skipped   1", "second build stdout")

    fbt(["build", "--project-dir", str(happy), "--select", "weekly_support_insights"])
    assert_file(happy / "target" / "artifacts" / "support" / "weekly_insights.md")
    return happy


def scenario_runner_failures(ctx: Context) -> None:
    capability_mismatch = ctx.tmpdir / "capability-mismatch"
    init_project(capability_mismatch)
    runner = capability_mismatch / "bin" / "fbt-demo-llm-runner"
    write(
        runner,
        f"""#!/usr/bin/env sh
export FBT_FAKE_RUNNER_ARTIFACT_TYPES=text
exec go run "{ROOT_DIR}/tests/runner_fixtures/fake" "$@"
""",
    )
    runner.chmod(0o755)
    result = fbt(["build", "--project-dir", str(capability_mismatch), "--select", "case_summaries"], expect=6)
    assert_contains(result.stderr, "runner capability incompatible", "capability mismatch stderr")
    run_results = (capability_mismatch / ".fbt" / "state" / "run_results.jsonl").read_text(encoding="utf-8")
    assert_regex(run_results, r'"record_type":"invocation_completed".*"status":"failed"', "capability run results")
    assert_contains(run_results, '"kind":"runner_capability_incompatible"', "capability run results")

    candidate_escape = ctx.tmpdir / "candidate-escape"
    init_project(candidate_escape)
    runner = candidate_escape / "bin" / "fbt-demo-llm-runner"
    write(
        runner,
        f"""#!/usr/bin/env sh
export FBT_FAKE_RUNNER_OUTPUT_OUTSIDE_WORK=1
exec go run "{ROOT_DIR}/tests/runner_fixtures/fake" "$@"
""",
    )
    runner.chmod(0o755)
    result = fbt(["build", "--project-dir", str(candidate_escape), "--select", "case_summaries"], expect=None)
    check(result.returncode != 0, "expected output candidate outside work dir to fail")
    assert_contains(result.stderr, "output candidate outside work outputs", "candidate escape stderr")
    assert_no_path(candidate_escape / "target" / "artifacts" / "support" / "case_summaries" / "index.md")
    run_results = (candidate_escape / ".fbt" / "state" / "run_results.jsonl").read_text(encoding="utf-8")
    assert_regex(run_results, r'"record_type":"invocation_completed".*"status":"failed"', "candidate run results")
    assert_contains(run_results, '"kind":"runner_contract_violation"', "candidate run results")


def write_runner_lock(project: Path, *, llm_command: str = "bin/fbt-demo-llm-runner", pin: str = "one") -> None:
    write(
        project / "fbt.lock.json",
        json.dumps(
            {
                "fbt_schema_version": "https://schemas.fbt.dev/fbt/runner-lock/v1.json",
                "lockfile_version": 1,
                "runners": {
                    "demo.llm": {
                        "protocol_version": "0.1",
                        "command": llm_command,
                        "capabilities": {
                            "transform_types": ["llm"],
                            "artifact_types": ["markdown_directory"],
                            "output_candidates": True,
                        },
                        "meta": {"pin": pin},
                    },
                    "demo.agent": {
                        "protocol_version": "0.1",
                        "command": "bin/fbt-demo-agent-runner",
                        "capabilities": {
                            "transform_types": ["agent"],
                            "artifact_types": ["markdown"],
                            "output_candidates": True,
                        },
                    },
                },
            },
            indent=2,
        )
        + "\n",
    )


def scenario_runner_lockfile(ctx: Context) -> None:
    locked = ctx.tmpdir / "runner-lock"
    init_project(locked)
    write_runner_lock(locked)
    doctor = fbt(["doctor", "--project-dir", str(locked)])
    assert_contains(doctor.stdout, "RUNNER_LOCK_OK", "runner lock doctor stdout")
    fbt(["build", "--project-dir", str(locked), "--select", "case_summaries"])

    write_runner_lock(locked, pin="two")
    plan = fbt(["plan", "--project-dir", str(locked), "--select", "case_summaries"])
    assert_contains(plan.stdout, "runner identity changed", "runner lock dirty plan")

    coverage = ctx.tmpdir / "runner-lock-coverage"
    init_project(coverage)
    write(
        coverage / "fbt.lock.json",
        json.dumps(
            {
                "fbt_schema_version": "https://schemas.fbt.dev/fbt/runner-lock/v1.json",
                "lockfile_version": 1,
                "runners": {"unused.runner": {"protocol_version": "0.1"}},
            },
            indent=2,
        )
        + "\n",
    )
    coverage_doctor = fbt(["doctor", "--project-dir", str(coverage)])
    assert_contains(coverage_doctor.stdout, "RUNNER_LOCK_MISSING", "runner lock coverage stdout")
    assert_contains(coverage_doctor.stdout, "RUNNER_LOCK_UNUSED", "runner lock coverage stdout")

    mismatch = ctx.tmpdir / "runner-lock-mismatch"
    init_project(mismatch)
    write_runner_lock(mismatch, llm_command="wrong-runner")
    result = fbt(["build", "--project-dir", str(mismatch), "--select", "case_summaries"], expect=6)
    assert_contains(result.stderr, "runner lock incompatible", "runner lock mismatch stderr")
    assert_no_path(mismatch / "target" / "artifacts" / "support" / "case_summaries" / "index.md")

    malformed = ctx.tmpdir / "runner-lock-malformed"
    init_project(malformed)
    write(
        malformed / "fbt.lock.json",
        '{"fbt_schema_version":"https://schemas.fbt.dev/fbt/runner-lock/v2.json","lockfile_version":2,"runners":{}}\n',
    )
    malformed_doctor = fbt(["doctor", "--project-dir", str(malformed)], expect=6)
    assert_contains(malformed_doctor.stdout, "RUNNER_LOCK_SCHEMA_UNSUPPORTED", "runner lock malformed stdout")


def scenario_orphaned_artifact_history(ctx: Context) -> None:
    project = ctx.tmpdir / "orphaned-artifact"
    init_project(project)
    fbt(["build", "--project-dir", str(project), "--select", "case_summaries"])

    write(project / "transforms" / "support" / "case_summaries.yml", "transforms: []\n")
    write(project / "transforms" / "support" / "weekly_insights.yml", "transforms: []\n")
    show = fbt(["artifact", "show", "case_summaries", "--project-dir", str(project)])
    assert_contains(show.stdout, "no (orphaned)", "orphaned artifact show")

    history = fbt(["artifact", "history", "case_summaries", "--project-dir", str(project)])
    assert_contains(history.stdout, "no (orphaned)", "orphaned artifact history")

    openlineage = fbt(["export", "openlineage", "--project-dir", str(project)])
    assert_contains(openlineage.stdout, '"orphaned":true', "orphaned OpenLineage export")
    assert_contains(openlineage.stdout, "artifact.", "orphaned OpenLineage export")


def validate_exports(openlineage_path: Path, otel_path: Path, redaction_marker: str) -> None:
    openlineage_text = openlineage_path.read_text(encoding="utf-8")
    otel_text = otel_path.read_text(encoding="utf-8")
    for label, text in [("openlineage", openlineage_text), ("otel", otel_text)]:
        if redaction_marker in text:
            fail(f"{label} export leaked redaction marker")
        if "Login issue resolved" in text:
            fail(f"{label} export leaked raw source content")

    events = [json.loads(line) for line in openlineage_text.splitlines() if line.strip()]
    check(len(events) >= 2, "expected at least two OpenLineage events")
    for event in events:
        check(event.get("eventType") == "COMPLETE", f"unexpected OpenLineage event type: {event}")
        check(
            event.get("schemaURL") == "https://openlineage.io/spec/1-0-0/OpenLineage.json#/definitions/RunEvent",
            "OpenLineage event missing schemaURL",
        )
        run_id = event.get("run", {}).get("runId", "")
        check(
            re.fullmatch(r"[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}", run_id) is not None,
            f"OpenLineage runId is not deterministic UUIDv5-shaped: {run_id}",
        )
        outputs = event.get("outputs", [])
        check(outputs and "fbt_artifact" in outputs[0].get("facets", {}), "OpenLineage output missing fbt_artifact facet")
        for dataset in event.get("inputs", []) + outputs:
            for key, facet in dataset.get("facets", {}).items():
                check(key.startswith("fbt_"), f"custom OpenLineage facet lacks fbt_ prefix: {key}")
                check(
                    str(facet.get("_schemaURL", "")).startswith("https://schemas.fbt.dev/openlineage/"),
                    f"custom OpenLineage facet lacks immutable fbt schema URL: {key}",
                )

    otel = json.loads(otel_text)
    resource_spans = otel.get("resourceSpans", [])
    check(resource_spans, "OTel export missing resourceSpans")
    spans: list[dict[str, object]] = []
    for resource_span in resource_spans:
        for scope_span in resource_span.get("scopeSpans", []):
            spans.extend(scope_span.get("spans", []))
    check(len(spans) >= 2, "expected OTel invocation and transform spans")

    def attr_keys(span: dict[str, object]) -> set[str]:
        return {attr.get("key") for attr in span.get("attributes", []) if isinstance(attr, dict)}

    check(any("fbt.invocation.id" in attr_keys(span) for span in spans), "OTel export missing invocation id attribute")
    check(any("fbt.transform.id" in attr_keys(span) for span in spans), "OTel export missing transform id attribute")
    check(any("gen_ai.usage.input_tokens" in attr_keys(span) for span in spans), "OTel export missing GenAI usage attribute")
    check(any(span.get("events") for span in spans), "OTel export missing runner span events")


def scenario_standard_exports(ctx: Context) -> None:
    happy = scenario_build_lifecycle(ctx)
    redaction_marker = "FBT_CONFORMANCE_SECRET_DO_NOT_EXPORT"
    openlineage = ctx.tmpdir / "openlineage.ndjson"
    otel = ctx.tmpdir / "otel.json"
    openlineage_again = ctx.tmpdir / "openlineage-again.ndjson"
    otel_again = ctx.tmpdir / "otel-again.json"

    fbt(["export", "openlineage", "--project-dir", str(happy), "--output", str(openlineage)])
    fbt(["export", "otel", "--project-dir", str(happy), "--output", str(otel)])
    fbt(["export", "openlineage", "--project-dir", str(happy), "--output", str(openlineage_again)])
    fbt(["export", "otel", "--project-dir", str(happy), "--output", str(otel_again)])

    check(openlineage.read_text(encoding="utf-8") == openlineage_again.read_text(encoding="utf-8"), "OpenLineage export is not deterministic")
    check(otel.read_text(encoding="utf-8") == otel_again.read_text(encoding="utf-8"), "OTel export is not deterministic")
    validate_exports(openlineage, otel, redaction_marker)

    append(happy / "assets" / "support_style_guide.md", "\n- Dirty propagation fixture\n")
    plan_dirty = fbt(["plan", "--project-dir", str(happy), "--select", "case_summaries"])
    assert_contains(plan_dirty.stdout, "RUN     case_summaries", "dirty plan stdout")


def scenario_policy_denied(ctx: Context) -> None:
    denied = ctx.tmpdir / "denied"
    init_project(denied)
    write(
        denied / "policies" / "support.yml",
        """policies:
  - name: support_agent_scope
    read: ["data/support/"]
    write: ["target/artifacts/other/"]
    network: false
""",
    )
    result = fbt(["build", "--project-dir", str(denied), "--select", "case_summaries"], expect=None)
    check(result.returncode != 0, "expected policy-denied build to fail")
    assert_no_path(denied / "target" / "artifacts" / "support" / "case_summaries" / "index.md")
    policy_decisions = (denied / ".fbt" / "state" / "policy_decisions.json").read_text(encoding="utf-8")
    assert_contains(policy_decisions, '"status": "denied"', "policy decisions")
    run_results = (denied / ".fbt" / "state" / "run_results.jsonl").read_text(encoding="utf-8")
    assert_regex(run_results, r'"record_type":"invocation_completed".*"status":"failed"', "denied run results")
    assert_contains(run_results, '"status":"policy_denied"', "denied run results")
    assert_contains(run_results, '"kind":"policy_denied"', "denied run results")


def scenario_agent_policy_required(ctx: Context) -> None:
    project = ctx.tmpdir / "agent-policy-required"
    init_project(project)
    transform_path = project / "transforms" / "support" / "weekly_insights.yml"
    content = transform_path.read_text(encoding="utf-8")
    transform_path.write_text(content.replace("    policy: support_agent_scope\n", ""), encoding="utf-8")

    result = fbt(["plan", "--project-dir", str(project), "--select", "weekly_insights"], expect=2)
    assert_contains(result.stderr, "AGENT_POLICY_MISSING", "agent missing policy stderr")
    assert_contains(result.stderr, "must declare an explicit policy", "agent missing policy stderr")


SCENARIOS = [
    Scenario("config version diagnostics", scenario_config_version),
    Scenario("strict YAML diagnostics", scenario_strict_yaml),
    Scenario("standard exports and dirty planning", scenario_standard_exports),
    Scenario("runner failure receipts", scenario_runner_failures),
    Scenario("runner lockfile diagnostics", scenario_runner_lockfile),
    Scenario("orphaned artifact history", scenario_orphaned_artifact_history),
    Scenario("policy denied receipts", scenario_policy_denied),
    Scenario("agent policy required", scenario_agent_policy_required),
]


def main() -> int:
    if not FBT_BIN.is_file() or not os.access(FBT_BIN, os.X_OK):
        print(f"FBT_BIN is not executable: {FBT_BIN}", file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as tmp:
        ctx = Context(tmpdir=Path(tmp))
        for scenario in SCENARIOS:
            try:
                scenario.run(ctx)
            except Exception as exc:  # noqa: BLE001 - test harness should report scenario context.
                print(f"conformance scenario failed: {scenario.name}", file=sys.stderr)
                print(str(exc), file=sys.stderr)
                return 1
    print("conformance: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
