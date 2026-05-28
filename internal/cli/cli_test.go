package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nyuta01/fbt/internal/artifact"
	"github.com/nyuta01/fbt/internal/state"
)

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "file build tool") {
		t.Fatalf("help output missing product description: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != "fbt "+Version {
		t.Fatalf("unexpected version output: %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunPlannedCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"build"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "not implemented yet") {
		t.Fatalf("expected not implemented message, got %q", stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"wat"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command: wat") {
		t.Fatalf("expected unknown command message, got %q", stderr.String())
	}
}

func TestRunParseWritesManifest(t *testing.T) {
	root := writeCLIProject(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"parse", "--project-dir", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Manifest written") {
		t.Fatalf("expected manifest output, got %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".fbt", "state", "manifest.json")); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
}

func TestRunPlanJSON(t *testing.T) {
	root := writeCLIProject(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"plan", "--project-dir", root, "--select", "tag:support", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"command": "plan"`) {
		t.Fatalf("expected JSON plan output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"action": "run"`) {
		t.Fatalf("expected run action, got %q", stdout.String())
	}
}

func TestRunParseMissingConfigVersion(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "fs_project.yml", "name: demo\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"parse", "--project-dir", root}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "CONFIG_VERSION_MISSING") {
		t.Fatalf("expected config version diagnostic, got %q", stderr.String())
	}
}

func TestRunStateAndArtifactCommands(t *testing.T) {
	root := writeCLIProject(t)
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	size := int64(7)
	version := state.ArtifactVersion{
		VersionID:   "artifact_version.knowledge_ops.case_summaries.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ArtifactID:  "artifact.knowledge_ops.case_summaries",
		LogicalPath: "target/artifacts/support/case_summaries/",
		StoragePath: "target/artifacts/support/case_summaries/",
		Descriptor: artifact.Descriptor{
			MediaType:    "text/markdown; charset=utf-8",
			Digest:       "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Size:         &size,
			ArtifactType: "fbt.artifact.markdown_document.v1",
		},
	}
	if err := store.PutArtifactVersion(version); err != nil {
		t.Fatal(err)
	}

	var stateOut bytes.Buffer
	var stateErr bytes.Buffer
	if code := Run([]string{"state", "status", "--project-dir", root}, &stateOut, &stateErr); code != 0 {
		t.Fatalf("state status failed: code=%d stderr=%q", code, stateErr.String())
	}
	if !strings.Contains(stateOut.String(), "Artifact versions: 1") {
		t.Fatalf("unexpected state status: %q", stateOut.String())
	}

	var artifactOut bytes.Buffer
	var artifactErr bytes.Buffer
	if code := Run([]string{"artifact", "versions", "case_summaries", "--project-dir", root}, &artifactOut, &artifactErr); code != 0 {
		t.Fatalf("artifact versions failed: code=%d stderr=%q", code, artifactErr.String())
	}
	if !strings.Contains(artifactOut.String(), version.VersionID) {
		t.Fatalf("unexpected artifact versions output: %q", artifactOut.String())
	}
}

func writeCLIProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
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
    command: fbt-openai-runner
selectors:
  - name: support_daily
    definition:
      method: tag
      value: support
`)
	writeFile(t, root, "data/support/tickets/2026-05-28.jsonl", "{}\n")
	writeFile(t, root, "prompts/case_summary.md", "# Task\n")
	writeFile(t, root, "sources/support.yml", `sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
        tags: ["support", "raw"]
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
    tags: ["support", "knowledge"]
`)
	return root
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
