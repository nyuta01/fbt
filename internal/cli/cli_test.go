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
	if got := strings.TrimSpace(stdout.String()); got != "fbt 0.1.0" {
		t.Fatalf("unexpected version output: %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunVersionJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"version", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"version": "0.1.0"`) {
		t.Fatalf("unexpected version JSON: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"commit": "unknown"`) {
		t.Fatalf("unexpected version JSON: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunPlannedCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"run"}, &stdout, &stderr)
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

func TestRunInitSupportTemplate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "knowledge_ops")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"init", root, "--template", "support"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Initialized support project") {
		t.Fatalf("unexpected init output: %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "fs_project.yml")); err != nil {
		t.Fatalf("expected project config: %v", err)
	}

	var parseOut bytes.Buffer
	var parseErr bytes.Buffer
	if code := Run([]string{"parse", "--project-dir", root}, &parseOut, &parseErr); code != 0 {
		t.Fatalf("generated project should parse: code=%d stderr=%q", code, parseErr.String())
	}

	var planOut bytes.Buffer
	var planErr bytes.Buffer
	if code := Run([]string{"plan", "--project-dir", root, "--select", "tag:support"}, &planOut, &planErr); code != 0 {
		t.Fatalf("generated project should plan: code=%d stderr=%q", code, planErr.String())
	}
	if !strings.Contains(planOut.String(), "next: fbt build --select case_summaries") {
		t.Fatalf("expected blocked next step, got %q", planOut.String())
	}

	var explainOut bytes.Buffer
	var explainErr bytes.Buffer
	if code := Run([]string{"artifact", "explain", "weekly_support_insights", "--project-dir", root}, &explainOut, &explainErr); code != 0 {
		t.Fatalf("generated project should explain artifact: code=%d stderr=%q", code, explainErr.String())
	}
	if !strings.Contains(explainOut.String(), "action: blocked") {
		t.Fatalf("expected blocked explanation, got %q", explainOut.String())
	}
	if !strings.Contains(explainOut.String(), "next: fbt build --select case_summaries") {
		t.Fatalf("expected explanation next step, got %q", explainOut.String())
	}

	var doctorOut bytes.Buffer
	var doctorErr bytes.Buffer
	if code := Run([]string{"doctor", "--project-dir", root}, &doctorOut, &doctorErr); code != 0 {
		t.Fatalf("generated project doctor failed: code=%d stdout=%q stderr=%q", code, doctorOut.String(), doctorErr.String())
	}
	if !strings.Contains(doctorOut.String(), "Doctor: ok") || !strings.Contains(doctorOut.String(), "RUNNER_PROTOCOL_OK") {
		t.Fatalf("expected doctor readiness output, got %q", doctorOut.String())
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
	if !strings.Contains(stderr.String(), "hint: Add `config_version: 1`") {
		t.Fatalf("expected config version hint, got %q", stderr.String())
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
		GeneratedBy: "transform_run.run_1",
		Confidence:  "structural",
		CommittedAt: "2026-05-28T00:00:00Z",
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

	var pathOut bytes.Buffer
	var pathErr bytes.Buffer
	if code := Run([]string{"artifact", "path", "case_summaries", "--project-dir", root}, &pathOut, &pathErr); code != 0 {
		t.Fatalf("artifact path failed: code=%d stderr=%q", code, pathErr.String())
	}
	if !strings.Contains(pathOut.String(), "logical_path: target/artifacts/support/case_summaries/") || !strings.Contains(pathOut.String(), "storage_path: target/artifacts/support/case_summaries/") {
		t.Fatalf("unexpected artifact path output: %q", pathOut.String())
	}

	var showOut bytes.Buffer
	var showErr bytes.Buffer
	if code := Run([]string{"artifact", "show", "case_summaries", "--project-dir", root}, &showOut, &showErr); code != 0 {
		t.Fatalf("artifact show failed: code=%d stderr=%q", code, showErr.String())
	}
	if !strings.Contains(showOut.String(), "generated_by: transform_run.run_1") || !strings.Contains(showOut.String(), "runner: runner.knowledge_ops.openai.responses") {
		t.Fatalf("unexpected artifact show output: %q", showOut.String())
	}

	var historyOut bytes.Buffer
	var historyErr bytes.Buffer
	if code := Run([]string{"artifact", "history", "case_summaries", "--project-dir", root}, &historyOut, &historyErr); code != 0 {
		t.Fatalf("artifact history failed: code=%d stderr=%q", code, historyErr.String())
	}
	if !strings.Contains(historyOut.String(), version.VersionID) || !strings.Contains(historyOut.String(), "committed_at: 2026-05-28T00:00:00Z") {
		t.Fatalf("unexpected artifact history output: %q", historyOut.String())
	}
}

func TestRunExportOpenLineage(t *testing.T) {
	root := writeCLIProject(t)
	store, version := writeCLICurrentArtifact(t, root)
	if err := store.PutApproval(state.Approval{
		ArtifactVersionID: version.VersionID,
		ArtifactID:        version.ArtifactID,
		Digest:            version.Descriptor.Digest,
		Status:            "approved",
		ReviewGroup:       "support_leads",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutEvaluationResult(state.EvaluationResult{
		ResultID:          "eval_result.knowledge_ops.case_summaries.required_case_sections",
		EvalID:            "eval.knowledge_ops.required_case_sections",
		ArtifactVersionID: version.VersionID,
		TransformRunID:    version.GeneratedBy,
		Status:            "pass",
		GrantsConfidence:  "structural",
	}); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "openlineage.ndjson")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"export", "openlineage", "--output", outputPath, "--project-dir", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("export openlineage failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "OpenLineage events written") || !strings.Contains(stdout.String(), "Events: 1") {
		t.Fatalf("unexpected export output: %q", stdout.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, expected := range []string{
		`"eventType":"COMPLETE"`,
		`"namespace":"fbt:knowledge_ops"`,
		`"name":"transform.knowledge_ops.case_summaries"`,
		`"fbt_artifact"`,
		`"fbt_approval"`,
		`"fbt_evaluations"`,
		version.VersionID,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected %q in OpenLineage export:\n%s", expected, content)
		}
	}
}

func TestRunExportOTel(t *testing.T) {
	root := writeCLIProject(t)
	store, version := writeCLICurrentArtifact(t, root)
	if err := store.AppendRunResult(map[string]any{
		"record_type":   "invocation_started",
		"invocation_id": "inv_cli",
		"started_at":    "2026-05-28T00:00:00Z",
		"command":       "build",
		"project_name":  "knowledge_ops",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendRunResult(map[string]any{
		"record_type":        "transform_run",
		"invocation_id":      "inv_cli",
		"run_id":             version.GeneratedBy,
		"transform_id":       "transform.knowledge_ops.case_summaries",
		"status":             "success",
		"started_at":         "2026-05-28T00:00:01Z",
		"completed_at":       "2026-05-28T00:00:02Z",
		"committed_versions": []string{version.VersionID},
		"usage": map[string]any{
			"gen_ai.usage.input_tokens":  10,
			"gen_ai.usage.output_tokens": 2,
			"fbt.usage.total_tokens":     12,
		},
		"events": []map[string]any{
			{
				"event_type": "usage",
				"time":       "2026-05-28T00:00:02Z",
				"attributes": map[string]any{"fbt.usage.total_tokens": 12},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendRunResult(map[string]any{
		"record_type":   "invocation_completed",
		"invocation_id": "inv_cli",
		"completed_at":  "2026-05-28T00:00:03Z",
		"status":        "success",
	}); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "otel.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"export", "otel", "--output", outputPath, "--project-dir", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("export otel failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "OTel traces written") || !strings.Contains(stdout.String(), "Spans: 2") {
		t.Fatalf("unexpected otel export output: %q", stdout.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, expected := range []string{
		`"resourceSpans"`,
		`"fbt.invocation.id"`,
		`"fbt.transform.id"`,
		`"gen_ai.usage.input_tokens"`,
		`"usage"`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected %q in OTel export:\n%s", expected, content)
		}
	}
}

func TestRunEvalAndReviewCommands(t *testing.T) {
	root := writeCLIProject(t)
	store, version := writeCLICurrentArtifact(t, root)
	if err := store.PutApproval(state.Approval{
		ArtifactVersionID: version.VersionID,
		ArtifactID:        version.ArtifactID,
		Digest:            version.Descriptor.Digest,
		Status:            "pending",
		ReviewGroup:       "support_leads",
	}); err != nil {
		t.Fatal(err)
	}

	var evalOut bytes.Buffer
	var evalErr bytes.Buffer
	if code := Run([]string{"eval", "case_summaries", "--project-dir", root}, &evalOut, &evalErr); code != 0 {
		t.Fatalf("eval failed: code=%d stdout=%q stderr=%q", code, evalOut.String(), evalErr.String())
	}
	if !strings.Contains(evalOut.String(), "pass eval.knowledge_ops.required_case_sections") {
		t.Fatalf("unexpected eval output: %q", evalOut.String())
	}

	var statusOut bytes.Buffer
	var statusErr bytes.Buffer
	if code := Run([]string{"review", "status", "case_summaries", "--project-dir", root}, &statusOut, &statusErr); code != 0 {
		t.Fatalf("review status failed: code=%d stdout=%q stderr=%q", code, statusOut.String(), statusErr.String())
	}
	if !strings.Contains(statusOut.String(), "status: pending") {
		t.Fatalf("unexpected review status: %q", statusOut.String())
	}
	if !strings.Contains(statusOut.String(), "next: fbt review show case_summaries") {
		t.Fatalf("expected review show guidance, got %q", statusOut.String())
	}

	var showOut bytes.Buffer
	var showErr bytes.Buffer
	if code := Run([]string{"review", "show", "case_summaries", "--project-dir", root}, &showOut, &showErr); code != 0 {
		t.Fatalf("review show failed: code=%d stdout=%q stderr=%q", code, showOut.String(), showErr.String())
	}
	if !strings.Contains(showOut.String(), "inspect: fbt artifact show case_summaries") {
		t.Fatalf("expected artifact inspection guidance, got %q", showOut.String())
	}
	if !strings.Contains(showOut.String(), "approve_after_review: fbt review approve case_summaries") {
		t.Fatalf("expected approval guidance, got %q", showOut.String())
	}

	var approveOut bytes.Buffer
	var approveErr bytes.Buffer
	if code := Run([]string{"review", "approve", "case_summaries", "--comment", "reviewed", "--project-dir", root}, &approveOut, &approveErr); code != 0 {
		t.Fatalf("review approve failed: code=%d stdout=%q stderr=%q", code, approveOut.String(), approveErr.String())
	}
	if !strings.Contains(approveOut.String(), "status: approved") || !strings.Contains(approveOut.String(), "confidence: reviewed") {
		t.Fatalf("unexpected approve output: %q", approveOut.String())
	}
	snapshot, err := store.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	pointer := snapshot.CurrentArtifacts[version.ArtifactID]
	if pointer.ApprovalStatus != "approved" || pointer.Confidence != "reviewed" {
		t.Fatalf("approval did not update pointer: %+v", pointer)
	}
}

func TestRunDiffAndDocsGenerate(t *testing.T) {
	root := writeCLIProject(t)
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	oldVersion := writeCLIArtifactVersion(t, store, root, "artifact_version.knowledge_ops.case_summaries.sha256_1111111111111111111111111111111111111111111111111111111111111111", "target/artifacts/support/case_summaries_old", "old")
	newVersion := writeCLIArtifactVersion(t, store, root, "artifact_version.knowledge_ops.case_summaries.sha256_2222222222222222222222222222222222222222222222222222222222222222", "target/artifacts/support/case_summaries_new", "new")
	if err := store.WriteState(state.Snapshot{
		CurrentArtifacts: map[string]state.ArtifactPointer{
			newVersion.ArtifactID: {
				ArtifactID:       newVersion.ArtifactID,
				CurrentVersionID: newVersion.VersionID,
				LogicalPath:      newVersion.LogicalPath,
				Confidence:       "structural",
				ApprovalStatus:   "pending",
			},
		},
		LatestRuns: map[string]state.LatestRun{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutApproval(state.Approval{
		ArtifactVersionID: oldVersion.VersionID,
		ArtifactID:        oldVersion.ArtifactID,
		Digest:            oldVersion.Descriptor.Digest,
		Status:            "approved",
		ReviewGroup:       "support_leads",
	}); err != nil {
		t.Fatal(err)
	}

	var diffOut bytes.Buffer
	var diffErr bytes.Buffer
	if code := Run([]string{"diff", "case_summaries", "--against", "last-approved", "--project-dir", root}, &diffOut, &diffErr); code != 0 {
		t.Fatalf("diff failed: code=%d stdout=%q stderr=%q", code, diffOut.String(), diffErr.String())
	}
	if !strings.Contains(diffOut.String(), "+new") || !strings.Contains(diffOut.String(), "changed: Summary") {
		t.Fatalf("unexpected diff output: %q", diffOut.String())
	}

	var docsOut bytes.Buffer
	var docsErr bytes.Buffer
	if code := Run([]string{"docs", "generate", "--project-dir", root}, &docsOut, &docsErr); code != 0 {
		t.Fatalf("docs generate failed: code=%d stdout=%q stderr=%q", code, docsOut.String(), docsErr.String())
	}
	docsPath := filepath.Join(root, "target", "docs", "index.md")
	data, err := os.ReadFile(docsPath)
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	if !strings.Contains(string(data), "artifact.knowledge_ops.case_summaries") {
		t.Fatalf("unexpected docs content: %s", string(data))
	}
}

func TestRunRunnerListAndDoctor(t *testing.T) {
	root := writeCLIProject(t)
	var listOut bytes.Buffer
	var listErr bytes.Buffer
	if code := Run([]string{"runner", "list", "--project-dir", root}, &listOut, &listErr); code != 0 {
		t.Fatalf("runner list failed: code=%d stderr=%q", code, listErr.String())
	}
	if !strings.Contains(listOut.String(), "openai.responses") {
		t.Fatalf("expected runner list output, got %q", listOut.String())
	}

	var doctorOut bytes.Buffer
	var doctorErr bytes.Buffer
	if code := Run([]string{"runner", "doctor", "openai.responses", "--project-dir", root}, &doctorOut, &doctorErr); code != 6 {
		t.Fatalf("expected missing runner exit code 6, got %d; stdout=%q stderr=%q", code, doctorOut.String(), doctorErr.String())
	}
	if !strings.Contains(doctorOut.String(), "RUNNER_COMMAND_NOT_FOUND") {
		t.Fatalf("expected missing command diagnostic, got %q", doctorOut.String())
	}

	var projectDoctorOut bytes.Buffer
	var projectDoctorErr bytes.Buffer
	if code := Run([]string{"doctor", "--project-dir", root}, &projectDoctorOut, &projectDoctorErr); code != 6 {
		t.Fatalf("expected project doctor exit code 6, got %d; stdout=%q stderr=%q", code, projectDoctorOut.String(), projectDoctorErr.String())
	}
	if !strings.Contains(projectDoctorOut.String(), "RUNNER_COMMAND_NOT_FOUND") {
		t.Fatalf("expected project doctor runner diagnostic, got %q", projectDoctorOut.String())
	}
}

func TestRunRunnerDoctorWithProjectCommand(t *testing.T) {
	root := writeCLIProject(t)
	command := filepath.Join(root, "bin", "fbt-openai-runner")
	writeFile(t, root, "bin/fbt-openai-runner", "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(command, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "fs_project.yml", strings.ReplaceAll(readFile(t, filepath.Join(root, "fs_project.yml")), "command: fbt-openai-runner", "command: bin/fbt-openai-runner"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"runner", "doctor", "openai.responses", "--project-dir", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("runner doctor failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "RUNNER_COMMAND_OK") {
		t.Fatalf("expected ok diagnostic, got %q", stdout.String())
	}
}

func writeCLIArtifactVersion(t *testing.T, store state.Store, root, versionID, storagePath, body string) state.ArtifactVersion {
	t.Helper()
	writeFile(t, root, filepath.Join(storagePath, "index.md"), "# Summary\n"+body+"\n")
	digest := strings.TrimPrefix(strings.TrimPrefix(versionID, "artifact_version.knowledge_ops.case_summaries."), "sha256_")
	version := state.ArtifactVersion{
		VersionID:   versionID,
		ArtifactID:  "artifact.knowledge_ops.case_summaries",
		LogicalPath: storagePath + "/",
		StoragePath: storagePath,
		Descriptor: artifact.Descriptor{
			MediaType:    "inode/directory",
			Digest:       "sha256:" + digest,
			ArtifactType: "fbt.artifact.markdown_directory.v1",
			FileCount:    1,
		},
	}
	if err := store.PutArtifactVersion(version); err != nil {
		t.Fatal(err)
	}
	return version
}

func writeCLICurrentArtifact(t *testing.T, root string) (state.Store, state.ArtifactVersion) {
	t.Helper()
	writeFile(t, root, "target/artifacts/support/case_summaries/index.md", "# Summary\n\nBody\n")
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	version := state.ArtifactVersion{
		VersionID:      "artifact_version.knowledge_ops.case_summaries.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ArtifactID:     "artifact.knowledge_ops.case_summaries",
		LogicalPath:    "target/artifacts/support/case_summaries/",
		StoragePath:    "target/artifacts/support/case_summaries/",
		GeneratedBy:    "transform_run.run_1",
		Confidence:     "structural",
		ApprovalStatus: "pending",
		Descriptor: artifact.Descriptor{
			MediaType:    "inode/directory",
			Digest:       "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			ArtifactType: "fbt.artifact.markdown_directory.v1",
			FileCount:    1,
		},
	}
	if err := store.PutArtifactVersion(version); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteState(state.Snapshot{
		CurrentArtifacts: map[string]state.ArtifactPointer{
			version.ArtifactID: {
				ArtifactID:       version.ArtifactID,
				CurrentVersionID: version.VersionID,
				CurrentDigest:    version.Descriptor.Digest,
				LogicalPath:      version.LogicalPath,
				Confidence:       "structural",
				ApprovalStatus:   "pending",
				GeneratedBy:      version.GeneratedBy,
			},
		},
		LatestRuns: map[string]state.LatestRun{},
	}); err != nil {
		t.Fatal(err)
	}
	return store, version
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
