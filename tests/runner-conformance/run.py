#!/usr/bin/env python3
"""Minimal fbt runner protocol conformance harness."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import queue
import shlex
import subprocess
import sys
import tempfile
import threading
import time


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run a minimal stdio JSON-RPC fbt runner conformance check."
    )
    parser.add_argument(
        "--runner-command",
        default=os.environ.get("FBT_RUNNER_CONFORMANCE_COMMAND", "go run ./runners/fake"),
        help="runner command to execute; defaults to FBT_RUNNER_CONFORMANCE_COMMAND or the source fake runner",
    )
    parser.add_argument(
        "--cwd",
        default=None,
        help="working directory for the runner command; defaults to the repository root",
    )
    parser.add_argument("--transform-type", default="llm")
    parser.add_argument("--artifact-type", default="markdown")
    parser.add_argument("--timeout-seconds", type=float, default=30.0)
    parser.add_argument(
        "--strict",
        action="store_true",
        help="require a progress event and fbt/outputCandidate notification, not only response outputs",
    )
    parser.add_argument(
        "--agent-adapter",
        action="store_true",
        help="require CLI-agent adapter safety markers and direct-write boundary checks",
    )
    return parser.parse_args()


def repo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def fail(message: str, stderr_path: Path | None = None) -> None:
    if stderr_path and stderr_path.exists():
        stderr_text = stderr_path.read_text(errors="replace").strip()
        if stderr_text:
            message = f"{message}\nrunner stderr:\n{stderr_text}"
    raise SystemExit(message)


def send(proc: subprocess.Popen[str], message: dict) -> None:
    if proc.stdin is None:
        fail("runner stdin is not available")
    proc.stdin.write(json.dumps(message, separators=(",", ":"), sort_keys=True) + "\n")
    proc.stdin.flush()


def enqueue_stdout(stream, output: queue.Queue[str | None]) -> None:
    try:
        for line in stream:
            output.put(line)
    finally:
        output.put(None)


def read_message(
    proc: subprocess.Popen[str],
    stdout_queue: queue.Queue[str | None],
    deadline: float,
    stderr_path: Path,
) -> dict:
    while time.monotonic() < deadline:
        timeout = max(0.0, min(0.2, deadline - time.monotonic()))
        try:
            line = stdout_queue.get(timeout=timeout)
        except queue.Empty:
            if proc.poll() is not None:
                fail(f"runner exited before response with code {proc.returncode}", stderr_path)
            continue
        if line is None:
            fail("runner closed stdout before response", stderr_path)
        try:
            message = json.loads(line)
        except json.JSONDecodeError as exc:
            fail(f"runner wrote invalid JSONL: {exc}: {line!r}", stderr_path)
        if not isinstance(message, dict):
            fail(f"runner message must be a JSON object: {message!r}", stderr_path)
        if message.get("jsonrpc") != "2.0":
            fail(f"runner message has invalid jsonrpc version: {message!r}", stderr_path)
        return message
    fail("timed out waiting for runner message", stderr_path)


def read_until_response(
    proc: subprocess.Popen[str],
    stdout_queue: queue.Queue[str | None],
    request_id: str,
    deadline: float,
    stderr_path: Path,
) -> tuple[dict, list[dict]]:
    notifications: list[dict] = []
    while True:
        try:
            message = read_message(proc, stdout_queue, deadline, stderr_path)
        except SystemExit as exc:
            fail(f"while waiting for {request_id}: {exc}", stderr_path)
        if message.get("id") == request_id:
            if "error" in message:
                fail(f"runner returned error for {request_id}: {message['error']}", stderr_path)
            if "result" not in message:
                fail(f"runner response missing result: {message!r}", stderr_path)
            return message, notifications
        if "method" in message and "id" not in message:
            notifications.append(message)
            continue
        fail(f"unexpected runner message while waiting for {request_id}: {message!r}", stderr_path)


def require(condition: bool, message: str, stderr_path: Path) -> None:
    if not condition:
        fail(message, stderr_path)


def list_capability(capabilities: dict, key: str) -> list[str]:
    value = capabilities.get(key)
    if not isinstance(value, list):
        return []
    return [item for item in value if isinstance(item, str)]


def within(parent: Path, child: Path) -> bool:
    try:
        return os.path.commonpath([str(parent), str(child)]) == str(parent)
    except ValueError:
        return False


def build_run_params(tmp: Path, args: argparse.Namespace) -> tuple[dict, dict[str, str | Path]]:
    project = tmp / "project"
    source_dir = project / "inputs"
    asset_dir = project / "assets"
    source_dir.mkdir(parents=True)
    asset_dir.mkdir(parents=True)
    source_path = source_dir / "source.md"
    asset_path = asset_dir / "prompt.md"
    redaction_marker = "FBT_RUNNER_CONFORMANCE_SECRET_DO_NOT_EXPORT"
    source_text = f"# Source\n\nA short fixture input.\n\nsecret={redaction_marker}\n"
    asset_text = f"# Prompt\n\nWrite a short fixture output. Do not leak {redaction_marker}.\n"
    source_path.write_text(source_text)
    asset_path.write_text(asset_text)

    work = project / ".fbt" / "work" / "runner-conformance"
    outputs = work / "outputs"
    (work / "tmp").mkdir(parents=True)
    outputs.mkdir(parents=True)

    logical_output = project / "target" / "artifacts" / "output.md"
    logical_output.parent.mkdir(parents=True)
    logical_output_text = "official output must not be touched by runner\n"
    logical_output.write_text(logical_output_text)
    state_path = project / ".fbt" / "state" / "state.json"
    state_path.parent.mkdir(parents=True)
    state_text = '{"guard":"state must not be touched by runner"}\n'
    state_path.write_text(state_text)

    params = {
        "mode": "run",
        "invocation_id": "invocation.runner_conformance",
        "transform_run_id": "transform_run.runner_conformance",
        "transform": {
            "unique_id": "transform.runner_conformance.output",
            "name": "output",
            "type": args.transform_type,
            "fingerprint": "sha256:runner-conformance",
        },
        "runner": {
            "name": "runner.conformance",
            "type": args.transform_type,
            "protocol": "stdio_jsonrpc",
            "env": [],
            "config": {},
        },
        "inputs": [
            {
                "kind": "source",
                "name": "source",
                "unique_id": "source.runner_conformance.source",
                "path": "inputs/source.md",
                "resolved_paths": [str(source_path)],
            }
        ],
        "outputs": [
            {
                "name": "output",
                "artifact_type": args.artifact_type,
                "declared_path": "target/artifacts/output.md",
            }
        ],
        "assets": [
            {
                "unique_id": "transform_asset.runner_conformance.prompt",
                "name": "prompt",
                "asset_type": "prompt",
                "path": "assets/prompt.md",
                "absolute_path": str(asset_path),
            }
        ],
        "model": {"provider": "conformance", "name": "fixture"},
        "tools": [],
        "policy": {
            "read": ["inputs/", "assets/"],
            "write": [".fbt/work/"],
            "network": False,
            "tools": {"allow": [], "deny": ["shell"]},
            "limits": {"timeout_seconds": int(args.timeout_seconds)},
        },
        "state": {
            "previous_run": {},
            "current_outputs": {},
            "plan": {"action": "run", "dirty_reasons": ["runner conformance"]},
        },
        "work": {
            "root": str(work),
            "temp": str(work / "tmp"),
            "outputs": str(outputs),
        },
    }
    guards: dict[str, str | Path] = {
        "project": project,
        "source_path": source_path,
        "source_text": source_text,
        "logical_output": logical_output,
        "logical_output_text": logical_output_text,
        "state_path": state_path,
        "state_text": state_text,
        "redaction_marker": redaction_marker,
    }
    return params, guards


def validate_initialize(response: dict, args: argparse.Namespace, stderr_path: Path) -> None:
    result = response["result"]
    require(isinstance(result, dict), "initialize result must be an object", stderr_path)
    require(isinstance(result.get("runner"), dict), "initialize result missing runner object", stderr_path)
    require(isinstance(result.get("protocol"), dict), "initialize result missing protocol object", stderr_path)
    require(
        result["protocol"].get("version") == "0.1",
        f"runner must negotiate protocol 0.1, got {result['protocol'].get('version')!r}",
        stderr_path,
    )
    capabilities = result.get("capabilities")
    require(isinstance(capabilities, dict), "initialize result missing capabilities object", stderr_path)
    transform_types = list_capability(capabilities, "transform_types")
    artifact_types = list_capability(capabilities, "artifact_types")
    require(
        args.transform_type in transform_types,
        f"capabilities.transform_types must include {args.transform_type!r}",
        stderr_path,
    )
    require(
        args.artifact_type in artifact_types,
        f"capabilities.artifact_types must include {args.artifact_type!r}",
        stderr_path,
    )
    require(
        capabilities.get("output_candidates") is not False,
        "capabilities.output_candidates must not be false",
        stderr_path,
    )


def validate_run(
    response: dict,
    notifications: list[dict],
    params: dict,
    args: argparse.Namespace,
    guards: dict[str, str | Path],
    stderr_path: Path,
) -> None:
    result = response["result"]
    require(isinstance(result, dict), "run result must be an object", stderr_path)
    require(result.get("status") == "success", f"expected success status, got {result!r}", stderr_path)
    require(
        result.get("transform_run_id") == params["transform_run_id"],
        "run result transform_run_id must match request",
        stderr_path,
    )

    event_count = sum(1 for message in notifications if message.get("method") == "fbt/event")
    candidate_messages = [
        message for message in notifications if message.get("method") == "fbt/outputCandidate"
    ]
    if args.strict:
        require(event_count > 0, "strict mode requires at least one fbt/event notification", stderr_path)
        require(
            len(candidate_messages) > 0,
            "strict mode requires at least one fbt/outputCandidate notification",
            stderr_path,
        )

    candidates: list[dict] = []
    for message in candidate_messages:
        outputs = message.get("params", {}).get("outputs", [])
        require(isinstance(outputs, list), "fbt/outputCandidate outputs must be a list", stderr_path)
        candidates.extend(output for output in outputs if isinstance(output, dict))
    if not candidates:
        outputs = result.get("outputs", [])
        require(isinstance(outputs, list), "run result outputs must be a list", stderr_path)
        candidates.extend(output for output in outputs if isinstance(output, dict))
    require(len(candidates) > 0, "runner must declare at least one output candidate", stderr_path)

    output_root = Path(params["work"]["outputs"]).resolve()
    for candidate in candidates:
        require(candidate.get("name") == "output", f"unexpected output name: {candidate!r}", stderr_path)
        if "artifact_type" in candidate:
            require(
                candidate["artifact_type"] == args.artifact_type,
                f"unexpected artifact_type: {candidate!r}",
                stderr_path,
            )
        path_value = candidate.get("path")
        require(isinstance(path_value, str) and path_value, f"candidate missing path: {candidate!r}", stderr_path)
        path = Path(path_value).resolve()
        require(path.exists(), f"candidate path does not exist: {path}", stderr_path)
        require(within(output_root, path), f"candidate path outside work.outputs: {path}", stderr_path)

    validate_redaction(response, notifications, guards, stderr_path)
    validate_direct_write_guards(guards, stderr_path)
    if args.agent_adapter:
        validate_agent_adapter_markers(notifications, params, stderr_path)


def validate_redaction(
    response: dict,
    notifications: list[dict],
    guards: dict[str, str | Path],
    stderr_path: Path,
) -> None:
    marker = str(guards["redaction_marker"])
    structured = json.dumps({"response": response, "notifications": notifications}, sort_keys=True)
    require(marker not in structured, "runner leaked source/asset redaction marker", stderr_path)


def validate_direct_write_guards(guards: dict[str, str | Path], stderr_path: Path) -> None:
    source_path = Path(guards["source_path"])
    logical_output = Path(guards["logical_output"])
    state_path = Path(guards["state_path"])
    require(source_path.read_text() == guards["source_text"], "runner modified source input", stderr_path)
    require(
        logical_output.read_text() == guards["logical_output_text"],
        "runner modified official artifact path directly",
        stderr_path,
    )
    require(state_path.read_text() == guards["state_text"], "runner modified .fbt/state directly", stderr_path)


def validate_agent_adapter_markers(
    notifications: list[dict],
    params: dict,
    stderr_path: Path,
) -> None:
    staging_paths: list[Path] = []
    policy_fail_closed = False
    for message in notifications:
        if message.get("method") != "fbt/event":
            continue
        attrs = message.get("params", {}).get("attributes", {})
        if not isinstance(attrs, dict):
            continue
        staging = attrs.get("fbt.adapter.staging_workspace")
        if isinstance(staging, str) and staging:
            staging_paths.append(Path(staging).resolve())
        if attrs.get("fbt.adapter.policy_mode") == "fail_closed":
            policy_fail_closed = True
        if attrs.get("fbt.adapter.policy_fail_closed") is True:
            policy_fail_closed = True

    work_root = Path(params["work"]["root"]).resolve()
    output_root = Path(params["work"]["outputs"]).resolve()
    require(staging_paths, "agent adapter must report fbt.adapter.staging_workspace", stderr_path)
    for staging in staging_paths:
        require(staging.exists(), f"reported staging workspace does not exist: {staging}", stderr_path)
        require(within(work_root, staging), f"staging workspace must be under work.root: {staging}", stderr_path)
        require(
            not within(output_root, staging),
            f"staging workspace must be separate from work.outputs: {staging}",
            stderr_path,
        )
    require(policy_fail_closed, "agent adapter must report fail-closed policy mapping", stderr_path)


def main() -> None:
    args = parse_args()
    root = repo_root()
    cwd = Path(args.cwd).resolve() if args.cwd else root
    argv = shlex.split(args.runner_command)
    if not argv:
        fail("runner command is empty")

    with tempfile.TemporaryDirectory(prefix="fbt-runner-conformance-") as tmpdir:
        tmp = Path(tmpdir)
        stderr_path = tmp / "runner.stderr"
        params, guards = build_run_params(tmp, args)
        env = os.environ.copy()
        with stderr_path.open("w+") as stderr_file:
            proc = subprocess.Popen(
                argv,
                cwd=str(cwd),
                env=env,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=stderr_file,
                text=True,
                bufsize=1,
            )
            if proc.stdout is None:
                fail("runner stdout is not available", stderr_path)
            stdout_queue: queue.Queue[str | None] = queue.Queue()
            threading.Thread(
                target=enqueue_stdout,
                args=(proc.stdout, stdout_queue),
                daemon=True,
            ).start()
            try:
                deadline = time.monotonic() + args.timeout_seconds
                send(
                    proc,
                    {
                        "jsonrpc": "2.0",
                        "id": "init_001",
                        "method": "initialize",
                        "params": {
                            "core": {"name": "fbt-runner-conformance", "version": "0.1.0"},
                            "protocol": {
                                "versions": ["0.1"],
                                "framing": "jsonl",
                                "schema_version": "https://schemas.fbt.dev/runner-protocol/v0.1.json",
                            },
                            "capability_request": [
                                "run_transform",
                                "stream_events",
                                "usage_reporting",
                                "output_candidates",
                                "cancellation",
                            ],
                        },
                    },
                )
                init_response, _ = read_until_response(
                    proc, stdout_queue, "init_001", deadline, stderr_path
                )
                validate_initialize(init_response, args, stderr_path)

                send(proc, {"jsonrpc": "2.0", "method": "initialized", "params": {}})
                deadline = time.monotonic() + args.timeout_seconds
                send(
                    proc,
                    {
                        "jsonrpc": "2.0",
                        "id": "run_001",
                        "method": "fbt/runTransform",
                        "params": params,
                    },
                )
                run_response, notifications = read_until_response(
                    proc, stdout_queue, "run_001", deadline, stderr_path
                )
                validate_run(run_response, notifications, params, args, guards, stderr_path)
            finally:
                if proc.stdin is not None:
                    try:
                        proc.stdin.close()
                    except BrokenPipeError:
                        pass
                try:
                    proc.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    proc.terminate()
                    try:
                        proc.wait(timeout=2)
                    except subprocess.TimeoutExpired:
                        proc.kill()
                        proc.wait(timeout=2)

    print("runner-conformance: ok")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        sys.exit(130)
