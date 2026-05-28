package build

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nyuta01/fbt/internal/state"
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
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	versions, err := store.ReadArtifactVersions()
	if err != nil {
		t.Fatalf("read artifact versions: %v", err)
	}
	version := versions.ArtifactVersions[result.Runs[0].CommittedVersions[0]]
	if version.SemanticDescriptor["text_normalized_v1"] == nil || version.SemanticDescriptor["markdown_ast_v1"] == nil {
		t.Fatalf("expected semantic descriptors, got %+v", version.SemanticDescriptor)
	}
	decisions, err := store.ReadPolicyDecisions()
	if err != nil {
		t.Fatalf("read policy decisions: %v", err)
	}
	if len(decisions.PolicyDecisions) != 1 {
		t.Fatalf("expected one policy decision, got %+v", decisions.PolicyDecisions)
	}
	for _, decision := range decisions.PolicyDecisions {
		if decision.Status != "allowed" || decision.ArtifactVersionID == "" || len(decision.Checks) == 0 {
			t.Fatalf("unexpected policy decision: %+v", decision)
		}
	}

	second, err := RunBuild(context.Background(), Options{ProjectDir: root, FBTVersion: "test"})
	if err != nil {
		t.Fatalf("run second build: %v", err)
	}
	if second.Plan.Summary.Skipped != 1 {
		t.Fatalf("expected clean second run to skip, got %+v", second.Plan.Summary)
	}
}

func TestRunBuildPolicyDenialDoesNotUpdateCurrentState(t *testing.T) {
	root := writeBuildProject(t)
	writeFile(t, root, "policies/support.yml", `policies:
  - name: support_agent_scope
    read: ["data/support/"]
    write: ["target/artifacts/other/"]
    network: true
`)
	_, err := RunBuild(context.Background(), Options{ProjectDir: root, FBTVersion: "test"})
	if err == nil {
		t.Fatal("expected policy denial")
	}
	if _, statErr := os.Stat(filepath.Join(root, "target", "artifacts", "support", "case_summaries", "index.md")); !os.IsNotExist(statErr) {
		t.Fatalf("official output should not be committed, stat err=%v", statErr)
	}
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	decisions, readErr := store.ReadPolicyDecisions()
	if readErr != nil {
		t.Fatalf("read policy decisions: %v", readErr)
	}
	if len(decisions.PolicyDecisions) != 1 {
		t.Fatalf("expected denied policy decision, got %+v", decisions.PolicyDecisions)
	}
	for _, decision := range decisions.PolicyDecisions {
		if decision.Status != "denied" || len(decision.Checks) == 0 {
			t.Fatalf("unexpected denied policy decision: %+v", decision)
		}
	}
}

func TestRunBuildRecordsEvalAndPendingReview(t *testing.T) {
	root := writeBuildProject(t)
	writeFile(t, root, "transforms/case.yml", strings.ReplaceAll(readFile(t, filepath.Join(root, "transforms", "case.yml")), `    tags: ["support"]
`, `    review:
      required: true
      group: support_leads
    tags: ["support"]
`))

	result, err := RunBuild(context.Background(), Options{ProjectDir: root, FBTVersion: "test"})
	if err != nil {
		t.Fatalf("run build: %v", err)
	}
	if len(result.Runs) != 1 || len(result.Runs[0].EvaluationResults) != 1 {
		t.Fatalf("expected evaluation result, got %+v", result.Runs)
	}
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	evals, err := store.ReadEvaluationResults()
	if err != nil {
		t.Fatalf("read eval results: %v", err)
	}
	if len(evals.EvaluationResults) != 1 {
		t.Fatalf("expected one eval result, got %+v", evals.EvaluationResults)
	}
	for _, result := range evals.EvaluationResults {
		if result.Status != "pass" || result.GrantsConfidence != "structural" {
			t.Fatalf("unexpected eval result: %+v", result)
		}
	}
	approvals, err := store.ReadApprovals()
	if err != nil {
		t.Fatalf("read approvals: %v", err)
	}
	if len(approvals.Approvals) != 1 {
		t.Fatalf("expected one pending approval, got %+v", approvals.Approvals)
	}
	for _, approval := range approvals.Approvals {
		if approval.Status != "pending" || approval.ReviewGroup != "support_leads" {
			t.Fatalf("unexpected approval: %+v", approval)
		}
	}
	snapshot, err := store.ReadState()
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	pointer := snapshot.CurrentArtifacts["artifact.knowledge_ops.case_summaries"]
	if pointer.ApprovalStatus != "pending" || pointer.Confidence != "structural" {
		t.Fatalf("unexpected current pointer: %+v", pointer)
	}
}

func TestRunBuildEvalFailureDoesNotCommitOutput(t *testing.T) {
	root := writeBuildProject(t)
	writeFile(t, root, "evals/support.yml", `evals:
  - name: required_case_sections
    type: deterministic
    config:
      sections: ["Missing"]
    grants_confidence: structural
`)
	_, err := RunBuild(context.Background(), Options{ProjectDir: root, FBTVersion: "test"})
	if err == nil {
		t.Fatal("expected eval failure")
	}
	if _, statErr := os.Stat(filepath.Join(root, "target", "artifacts", "support", "case_summaries", "index.md")); !os.IsNotExist(statErr) {
		t.Fatalf("official output should not be committed, stat err=%v", statErr)
	}
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	evals, readErr := store.ReadEvaluationResults()
	if readErr != nil {
		t.Fatalf("read eval results: %v", readErr)
	}
	if len(evals.EvaluationResults) != 1 {
		t.Fatalf("expected failed eval result, got %+v", evals.EvaluationResults)
	}
	for _, result := range evals.EvaluationResults {
		if result.Status != "fail" {
			t.Fatalf("unexpected eval result: %+v", result)
		}
	}
}

func TestRunBuildWithLocalLLMRunnerRecordsUsageAndProvenance(t *testing.T) {
	root := writeBuildProject(t)
	repoRoot := repoRoot(t)
	writeFile(t, root, "bin/fbt-local-llm-runner", "#!/bin/sh\nexec go run "+shellQuote(filepath.Join(repoRoot, "runners", "llm"))+"\n")
	if err := os.Chmod(filepath.Join(root, "bin", "fbt-local-llm-runner"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "fs_project.yml", strings.ReplaceAll(readFile(t, filepath.Join(root, "fs_project.yml")), "command: bin/fbt-fake-runner", "command: bin/fbt-local-llm-runner"))
	writeFile(t, root, "evals/support.yml", `evals:
  - name: required_case_sections
    type: deterministic
    config:
      sections: ["Case Summaries"]
    grants_confidence: structural
`)
	writeFile(t, root, "transforms/case.yml", strings.ReplaceAll(readFile(t, filepath.Join(root, "transforms", "case.yml")), `    runner: openai.responses
`, `    runner: openai.responses
    model:
      provider: local
      name: build-mock-gpt
`))

	if _, err := RunBuild(context.Background(), Options{ProjectDir: root, FBTVersion: "test"}); err != nil {
		t.Fatalf("run build: %v", err)
	}
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	records, err := store.ReadRunResults()
	if err != nil {
		t.Fatalf("read run results: %v", err)
	}
	var found bool
	for _, record := range records {
		if record["record_type"] != "transform_run" {
			continue
		}
		usage, usageOK := record["usage"].(map[string]any)
		provenance, provenanceOK := record["provenance"].(map[string]any)
		if usageOK && usage["fbt.usage.total_tokens"] != nil && provenanceOK && provenance["model"] == "build-mock-gpt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected usage/provenance transform run record, got %+v", records)
	}
}

func TestRunBuildStoresImmutableVersionContent(t *testing.T) {
	root := writeBuildProject(t)
	repoRoot := repoRoot(t)
	writeFile(t, root, "bin/fbt-local-llm-runner", "#!/bin/sh\nexec go run "+shellQuote(filepath.Join(repoRoot, "runners", "llm"))+"\n")
	if err := os.Chmod(filepath.Join(root, "bin", "fbt-local-llm-runner"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "fs_project.yml", strings.ReplaceAll(readFile(t, filepath.Join(root, "fs_project.yml")), "command: bin/fbt-fake-runner", "command: bin/fbt-local-llm-runner"))
	writeFile(t, root, "evals/support.yml", `evals:
  - name: required_case_sections
    type: deterministic
    config:
      sections: ["Case Summaries"]
    grants_confidence: structural
`)
	transformPath := filepath.Join(root, "transforms", "case.yml")
	writeFile(t, root, "transforms/case.yml", strings.ReplaceAll(readFile(t, transformPath), `    runner: openai.responses
`, `    runner: openai.responses
    model:
      provider: local
      name: first-model
`))
	if _, err := RunBuild(context.Background(), Options{ProjectDir: root, FBTVersion: "test"}); err != nil {
		t.Fatalf("first build: %v", err)
	}
	writeFile(t, root, "transforms/case.yml", strings.ReplaceAll(readFile(t, transformPath), "name: first-model", "name: second-model"))
	if _, err := RunBuild(context.Background(), Options{ProjectDir: root, FBTVersion: "test"}); err != nil {
		t.Fatalf("second build: %v", err)
	}

	store := state.Open(filepath.Join(root, ".fbt", "state"))
	versions, err := store.ReadArtifactVersions()
	if err != nil {
		t.Fatal(err)
	}
	if len(versions.ArtifactVersions) != 2 {
		t.Fatalf("expected two artifact versions, got %+v", versions.ArtifactVersions)
	}
	var sawFirst, sawSecond bool
	for _, version := range versions.ArtifactVersions {
		if strings.HasPrefix(version.StoragePath, "target/artifacts/") {
			t.Fatalf("version storage path should be immutable internal storage: %+v", version)
		}
		data, err := os.ReadFile(filepath.Join(root, version.StoragePath, "index.md"))
		if err != nil {
			t.Fatalf("read stored version content: %v", err)
		}
		sawFirst = sawFirst || strings.Contains(string(data), "first-model")
		sawSecond = sawSecond || strings.Contains(string(data), "second-model")
	}
	if !sawFirst || !sawSecond {
		t.Fatalf("expected both stored version contents, first=%v second=%v", sawFirst, sawSecond)
	}
}

func TestRunBuildPassesCompleteProtocolContext(t *testing.T) {
	root := writeBuildProject(t)
	repoRoot := repoRoot(t)
	writeFile(t, root, "fs_project.yml", strings.ReplaceAll(readFile(t, filepath.Join(root, "fs_project.yml")), `    command: bin/fbt-fake-runner
`, `    command: bin/fbt-fake-runner
    env:
      - FBT_FAKE_RUNNER_CAPTURE_PARAMS
    config:
      provider: test
      default_model: fake
`))
	writeFile(t, root, "sources/support.yml", strings.ReplaceAll(readFile(t, filepath.Join(root, "sources", "support.yml")), "path: data/support/tickets/*.jsonl", "path: data/support/tickets/"))
	writeFile(t, root, "transforms/weekly.yml", `transforms:
  - name: weekly_report
    type: llm
    runner: openai.responses
    inputs:
      - source: support.raw_tickets
      - ref: case_summaries
    outputs:
      - name: weekly_report
        type: markdown
        path: target/artifacts/support/weekly_report.md
    assets:
      - ref: case_summary_prompt
    tags: ["support"]
`)

	if _, err := RunBuild(context.Background(), Options{ProjectDir: root, Select: "case_summaries", FBTVersion: "test"}); err != nil {
		t.Fatalf("first build: %v", err)
	}

	capturePath := filepath.Join(root, ".fbt", "captured-runner-params.json")
	t.Setenv("FBT_FAKE_RUNNER_CAPTURE_PARAMS", capturePath)
	writeFile(t, root, "bin/fbt-fake-runner", "#!/bin/sh\nexec go run "+shellQuote(filepath.Join(repoRoot, "runners", "fake"))+"\n")
	if err := os.Chmod(filepath.Join(root, "bin", "fbt-fake-runner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := RunBuild(context.Background(), Options{ProjectDir: root, Select: "weekly_report", FBTVersion: "test"}); err != nil {
		t.Fatalf("second build: %v", err)
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(readFile(t, capturePath)), &params); err != nil {
		t.Fatalf("decode captured params: %v", err)
	}
	inputs := params["inputs"].([]any)
	if len(inputs) != 2 {
		t.Fatalf("expected source and ref inputs, got %+v", inputs)
	}
	sourceInput := inputs[0].(map[string]any)
	if sourceInput["kind"] != "source" || sourceInput["descriptor"] == nil || len(sourceInput["resolved_paths"].([]any)) == 0 {
		t.Fatalf("expected resolved source descriptor, got %+v", sourceInput)
	}
	refInput := inputs[1].(map[string]any)
	currentVersion := refInput["current_version"].(map[string]any)
	if refInput["kind"] != "ref" || currentVersion["version_id"] == "" || currentVersion["descriptor"] == nil || currentVersion["semantic_descriptor"] == nil {
		t.Fatalf("expected current artifact version context, got %+v", refInput)
	}
	assets := params["assets"].([]any)
	if len(assets) != 1 {
		t.Fatalf("expected one asset, got %+v", assets)
	}
	asset := assets[0].(map[string]any)
	if asset["name"] != "case_summary_prompt" || asset["absolute_path"] == "" || asset["fingerprint"] == nil {
		t.Fatalf("expected asset context, got %+v", asset)
	}
	runner := params["runner"].(map[string]any)
	config := runner["config"].(map[string]any)
	if runner["name"] != "openai.responses" || config["provider"] != "test" {
		t.Fatalf("expected runner config context, got %+v", runner)
	}
	statePayload := params["state"].(map[string]any)
	plan := statePayload["plan"].(map[string]any)
	if plan["action"] != "run" || len(plan["dirty_reasons"].([]any)) == 0 {
		t.Fatalf("expected plan state context, got %+v", statePayload)
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
      sections: ["Fake Output"]
    grants_confidence: structural
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
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
