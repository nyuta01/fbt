package build

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunBuildCommitsFakeRunnerOutputAndSkipsCleanSecondRun(t *testing.T) {
	root := writeBuildProject(t)
	result, err := RunBuild(context.Background(), Options{ProjectDir: root, FBTVersion: "test"})
	if err != nil {
		t.Fatalf("run build: %v", err)
	}
	if result.Plan.Summary.Run != 1 {
		t.Fatalf("expected one run, got %+v", result.Plan.Summary)
	}
	if len(result.Runs) != 1 || len(result.Runs[0].CommittedVersions) != 1 {
		t.Fatalf("expected committed version, got %+v", result.Runs)
	}
	if _, err := os.Stat(filepath.Join(root, "target", "artifacts", "support", "case_summaries", "index.md")); err != nil {
		t.Fatalf("expected committed output: %v", err)
	}

	second, err := RunBuild(context.Background(), Options{ProjectDir: root, FBTVersion: "test"})
	if err != nil {
		t.Fatalf("run second build: %v", err)
	}
	if second.Plan.Summary.Skipped != 1 {
		t.Fatalf("expected clean second run to skip, got %+v", second.Plan.Summary)
	}
}

func writeBuildProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repoRoot := repoRoot(t)
	writeFile(t, root, "bin/fbt-fake-runner", "#!/bin/sh\nexec go run "+shellQuote(filepath.Join(repoRoot, "runners", "fake"))+"\n")
	if err := os.Chmod(filepath.Join(root, "bin", "fbt-fake-runner"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "fs_project.yml", `name: knowledge_ops
config_version: 1
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
artifact_path: "target/artifacts"
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: bin/fbt-fake-runner
`)
	writeFile(t, root, "data/support/tickets/2026-05-28.jsonl", "{}\n")
	writeFile(t, root, "prompts/case_summary.md", "# Task\n")
	writeFile(t, root, "sources/support.yml", `sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
        tags: ["support"]
`)
	writeFile(t, root, "assets/assets.yml", `assets:
  - name: case_summary_prompt
    type: prompt
    path: prompts/case_summary.md
`)
	writeFile(t, root, "policies/support.yml", `policies:
  - name: support_agent_scope
    read: ["data/support/"]
    write: [".fbt/work/", "target/artifacts/support/"]
    network: true
`)
	writeFile(t, root, "evals/support.yml", `evals:
  - name: required_case_sections
    type: deterministic
    config:
      sections: ["Summary"]
`)
	writeFile(t, root, "transforms/case.yml", `transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    assets:
      - ref: case_summary_prompt
    policy: support_agent_scope
    evals:
      - required_case_sections
    tags: ["support"]
`)
	return root
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func shellQuote(value string) string {
	return "'" + value + "'"
}

func writeFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
