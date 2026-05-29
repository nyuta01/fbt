package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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
	if !strings.Contains(stdout.String(), "versioned filesystem artifacts") {
		t.Fatalf("help output missing product description: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunHelpDescribesCommandOutputs(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name: "root flags",
			args: []string{"--help"},
			expected: []string{
				"builds versioned filesystem artifacts",
				"Select transforms for plan/build",
				"immutable artifact storage stays under .fbt/artifacts",
			},
		},
		{
			name: "artifact subcommands",
			args: []string{"artifact", "--help"},
			expected: []string{
				"Show the current artifact version and metadata",
				"Explain why an artifact will run, skip, or block",
				"Report local state and artifact storage usage",
			},
		},
		{
			name: "standard exports",
			args: []string{"export", "--help"},
			expected: []string{
				"RunEvent NDJSON",
				"OTLP/JSON execution traces",
				"stdout for normal shell piping",
			},
		},
		{
			name: "openlineage details",
			args: []string{"export", "openlineage", "--help"},
			expected: []string{
				"OpenLineage-compatible RunEvent NDJSON",
				"written to stdout",
				"lineage backend",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if code := Run(tc.args, &stdout, &stderr); code != 0 {
				t.Fatalf("expected exit code 0, got %d; stderr=%q", code, stderr.String())
			}
			for _, expected := range tc.expected {
				if !strings.Contains(stdout.String(), expected) {
					t.Fatalf("expected %q in help output:\n%s", expected, stdout.String())
				}
			}
			if stderr.Len() != 0 {
				t.Fatalf("expected empty stderr, got %q", stderr.String())
			}
		})
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

func TestRunFormerPlaceholderCommandIsUnknown(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"run"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), `unknown command "run"`) {
		t.Fatalf("expected unknown command message, got %q", stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"wat"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), `unknown command "wat"`) {
		t.Fatalf("expected unknown command message, got %q", stderr.String())
	}
}

func TestRunRejectsUnknownAndExtraArguments(t *testing.T) {
	root := writeCLIProject(t)

	cases := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "plan unknown flag",
			args:     []string{"plan", "--bogus", "--project-dir", root},
			expected: "unknown flag: --bogus",
		},
		{
			name:     "build extra arg",
			args:     []string{"build", "case_summaries", "--project-dir", root},
			expected: `unknown command "case_summaries"`,
		},
		{
			name:     "artifact extra flag",
			args:     []string{"artifact", "show", "case_summaries", "--bogus", "--project-dir", root},
			expected: "unknown flag: --bogus",
		},
		{
			name:     "artifact history extra arg",
			args:     []string{"artifact", "history", "case_summaries", "extra", "--project-dir", root},
			expected: "accepts 1 arg(s), received 2",
		},
		{
			name:     "export extra arg",
			args:     []string{"export", "openlineage", "extra", "--project-dir", root},
			expected: `unknown command "extra"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if code := Run(tc.args, &stdout, &stderr); code != 2 {
				t.Fatalf("expected exit code 2, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.expected) {
				t.Fatalf("expected %q in stderr, got %q", tc.expected, stderr.String())
			}
		})
	}
}

func TestRunSelectNoMatchFails(t *testing.T) {
	root := writeCLIProject(t)

	var planOut bytes.Buffer
	var planErr bytes.Buffer
	if code := Run([]string{"plan", "--select", "no_such", "--project-dir", root}, &planOut, &planErr); code != 2 {
		t.Fatalf("expected plan exit code 2, got %d; stdout=%q stderr=%q", code, planOut.String(), planErr.String())
	}
	if !strings.Contains(planErr.String(), "selector matched no transforms: no_such") {
		t.Fatalf("expected selector error, got %q", planErr.String())
	}

	var buildOut bytes.Buffer
	var buildErr bytes.Buffer
	if code := Run([]string{"build", "--select", "no_such", "--project-dir", root}, &buildOut, &buildErr); code != 2 {
		t.Fatalf("expected build exit code 2, got %d; stdout=%q stderr=%q", code, buildOut.String(), buildErr.String())
	}
	if !strings.Contains(buildErr.String(), "selector matched no transforms: no_such") {
		t.Fatalf("expected selector error, got %q", buildErr.String())
	}
}

func TestRunActionableErrorHints(t *testing.T) {
	root := writeCLIProject(t)

	cases := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name: "declared artifact has no version",
			args: []string{"artifact", "show", "case_summaries", "--project-dir", root},
			expected: []string{
				"artifact has no built version yet: case_summaries",
				"Hint: run `fbt build --select case_summaries` to create it.",
			},
		},
		{
			name: "empty selector",
			args: []string{"plan", "--select", "no_such", "--project-dir", root},
			expected: []string{
				"selector matched no transforms: no_such",
				"Hint: run `fbt plan` without --select",
			},
		},
		{
			name: "dry run flag",
			args: []string{"build", "--dry-run", "--project-dir", root},
			expected: []string{
				"unknown flag: --dry-run",
				"Hint: use `fbt plan` to preview without writing state or starting runners.",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if code := Run(tc.args, &stdout, &stderr); code != 2 {
				t.Fatalf("expected exit code 2, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			for _, expected := range tc.expected {
				if !strings.Contains(stderr.String(), expected) {
					t.Fatalf("expected %q in stderr:\n%s", expected, stderr.String())
				}
			}
		})
	}
}

func TestRunPlanIsReadOnly(t *testing.T) {
	root := writeCLIProject(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"plan", "--project-dir", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Plan\n") || !strings.Contains(stdout.String(), "selected  1") {
		t.Fatalf("expected plan output, got %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".fbt", "state", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("plan should not write manifest, stat err=%v", err)
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
	if !strings.Contains(stdout.String(), "Demo runners: configured as demo.*") {
		t.Fatalf("expected demo runner hint, got %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "fs_project.yml")); err != nil {
		t.Fatalf("expected project config: %v", err)
	}

	var planOut bytes.Buffer
	var planErr bytes.Buffer
	if code := Run([]string{"plan", "--project-dir", root, "--select", "tag:support"}, &planOut, &planErr); code != 0 {
		t.Fatalf("generated project should plan: code=%d stderr=%q", code, planErr.String())
	}
	if !strings.Contains(planOut.String(), "next     fbt build --select case_summaries --project-dir "+shellArg(root)) {
		t.Fatalf("expected blocked next step, got %q", planOut.String())
	}

	var explainOut bytes.Buffer
	var explainErr bytes.Buffer
	if code := Run([]string{"artifact", "explain", "weekly_support_insights", "--project-dir", root}, &explainOut, &explainErr); code != 0 {
		t.Fatalf("generated project should explain artifact: code=%d stderr=%q", code, explainErr.String())
	}
	if !strings.Contains(explainOut.String(), "Decision: BLOCK") {
		t.Fatalf("expected blocked explanation, got %q", explainOut.String())
	}
	if !strings.Contains(explainOut.String(), "Reason  requires case_summaries current artifact") {
		t.Fatalf("expected decision explanation, got %q", explainOut.String())
	}
	if !strings.Contains(explainOut.String(), "missing  input") || !strings.Contains(explainOut.String(), "case_summaries") {
		t.Fatalf("expected upstream artifact input detail, got %q", explainOut.String())
	}
	if !strings.Contains(explainOut.String(), "fbt build --select case_summaries --project-dir "+shellArg(root)) {
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

func TestRunNextCommandsPreserveInvocationContext(t *testing.T) {
	root := writeCLIProject(t)
	stateDir := filepath.Join(t.TempDir(), "custom state")
	expectedNext := "fbt build --select case_summaries --project-dir " + shellArg(root) + " --state-dir " + shellArg(stateDir)

	var planOut bytes.Buffer
	var planErr bytes.Buffer
	if code := Run([]string{"plan", "--project-dir", root, "--state-dir", stateDir, "--select", "case_summaries"}, &planOut, &planErr); code != 0 {
		t.Fatalf("plan failed: code=%d stdout=%q stderr=%q", code, planOut.String(), planErr.String())
	}
	if !strings.Contains(planOut.String(), expectedNext) {
		t.Fatalf("expected contextual next step %q, got %q", expectedNext, planOut.String())
	}

	var explainOut bytes.Buffer
	var explainErr bytes.Buffer
	if code := Run([]string{"artifact", "explain", "case_summaries", "--project-dir", root, "--state-dir", stateDir}, &explainOut, &explainErr); code != 0 {
		t.Fatalf("explain failed: code=%d stdout=%q stderr=%q", code, explainOut.String(), explainErr.String())
	}
	if !strings.Contains(explainOut.String(), expectedNext) {
		t.Fatalf("expected contextual explanation next step %q, got %q", expectedNext, explainOut.String())
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

func TestRunPlanForceShowsForcedRebuild(t *testing.T) {
	root := writeCLIProject(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"plan", "--project-dir", root, "--force"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "because  forced rebuild") {
		t.Fatalf("expected forced rebuild reason, got %q", stdout.String())
	}
}

func TestRunBuildShowsCommittedArtifactPathAndNext(t *testing.T) {
	root := writeCLIProject(t)
	repoRoot := repoRoot(t)
	command := filepath.Join(root, "bin", "fbt-openai-runner")
	writeFile(t, root, "bin/fbt-openai-runner", "#!/bin/sh\nexec go run "+shellQuote(filepath.Join(repoRoot, "runners", "fake"))+"\n")
	if err := os.Chmod(command, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "fs_project.yml", strings.ReplaceAll(readFile(t, filepath.Join(root, "fs_project.yml")), "command: fbt-openai-runner", "command: bin/fbt-openai-runner"))
	writeFile(t, root, "evals/support.yml", `evals:
  - name: required_case_sections
    type: deterministic
    config:
      sections: ["Fake Output"]
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"build", "--project-dir", root, "--select", "case_summaries"}, &stdout, &stderr); code != 0 {
		t.Fatalf("build failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	expectedOutput := "output     case_summaries -> target/artifacts/support/case_summaries"
	if !strings.Contains(stdout.String(), expectedOutput) {
		t.Fatalf("expected committed output path %q, got %q", expectedOutput, stdout.String())
	}
	expectedNext := "next       fbt artifact show case_summaries --project-dir " + shellArg(root)
	if !strings.Contains(stdout.String(), expectedNext) {
		t.Fatalf("expected artifact inspection next step %q, got %q", expectedNext, stdout.String())
	}
}

func TestRunPlanMissingConfigVersion(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "fs_project.yml", "name: demo\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"plan", "--project-dir", root}, &stdout, &stderr)
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

func TestRunArtifactCommands(t *testing.T) {
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

	var pathOut bytes.Buffer
	var pathErr bytes.Buffer
	if code := Run([]string{"artifact", "path", "case_summaries", "--project-dir", root}, &pathOut, &pathErr); code != 0 {
		t.Fatalf("artifact path failed: code=%d stderr=%q", code, pathErr.String())
	}
	if !strings.Contains(pathOut.String(), "Logical path    target/artifacts/support/case_summaries/") || !strings.Contains(pathOut.String(), "Immutable path  target/artifacts/support/case_summaries/") {
		t.Fatalf("unexpected artifact path output: %q", pathOut.String())
	}

	var showOut bytes.Buffer
	var showErr bytes.Buffer
	if code := Run([]string{"artifact", "show", "case_summaries", "--project-dir", root}, &showOut, &showErr); code != 0 {
		t.Fatalf("artifact show failed: code=%d stderr=%q", code, showErr.String())
	}
	if !strings.Contains(showOut.String(), "Run        transform_run.run_1") || !strings.Contains(showOut.String(), "Runner     openai.responses") {
		t.Fatalf("unexpected artifact show output: %q", showOut.String())
	}

	var historyOut bytes.Buffer
	var historyErr bytes.Buffer
	if code := Run([]string{"artifact", "history", "case_summaries", "--project-dir", root}, &historyOut, &historyErr); code != 0 {
		t.Fatalf("artifact history failed: code=%d stderr=%q", code, historyErr.String())
	}
	if !strings.Contains(historyOut.String(), version.VersionID) || !strings.Contains(historyOut.String(), "Committed  2026-05-28T00:00:00Z") {
		t.Fatalf("unexpected artifact history output: %q", historyOut.String())
	}

	var versionsOut bytes.Buffer
	var versionsErr bytes.Buffer
	if code := Run([]string{"artifact", "versions", "case_summaries", "--project-dir", root}, &versionsOut, &versionsErr); code != 2 {
		t.Fatalf("artifact versions should be pruned: code=%d stdout=%q stderr=%q", code, versionsOut.String(), versionsErr.String())
	}
	if !strings.Contains(versionsErr.String(), `unknown command "versions"`) {
		t.Fatalf("expected unknown artifact command, got %q", versionsErr.String())
	}
}

func TestRunArtifactRetentionReportsReadOnlyHygiene(t *testing.T) {
	root := writeCLIProject(t)
	store, current := writeCLICurrentArtifact(t, root)
	writeCLIArtifactVersion(t, store, root, "artifact_version.knowledge_ops.case_summaries.sha256_1111111111111111111111111111111111111111111111111111111111111111", ".fbt/artifacts/historical/content", "old")
	if err := store.AppendRunResult(map[string]any{"record_type": "invocation_started", "invocation_id": "inv_cli"}); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"artifact", "retention", "--project-dir", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("artifact retention failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, expected := range []string{
		"Artifact retention",
		"Policy               keep_all",
		"Artifact versions    2",
		"Current versions     1",
		"Historical versions  1",
		"Run records          1",
		"Action               no files removed",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected %q in retention output:\n%s", expected, out)
		}
	}

	var jsonOut bytes.Buffer
	var jsonErr bytes.Buffer
	if code := Run([]string{"artifact", "retention", "--json", "--project-dir", root}, &jsonOut, &jsonErr); code != 0 {
		t.Fatalf("artifact retention json failed: code=%d stderr=%q", code, jsonErr.String())
	}
	if !strings.Contains(jsonOut.String(), current.VersionID) {
		t.Fatalf("expected current version context in json output: %s", jsonOut.String())
	}
	if !strings.Contains(jsonOut.String(), `"historical_versions": 1`) {
		t.Fatalf("expected historical count in json output: %s", jsonOut.String())
	}
}

func TestRunExportOpenLineage(t *testing.T) {
	root := writeCLIProject(t)
	store, version := writeCLICurrentArtifact(t, root)
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

func TestRunPrunedCommandsAreUnknown(t *testing.T) {
	root := writeCLIProject(t)
	for _, command := range []string{"parse", "eval", "docs", "state", "runner", "review"} {
		t.Run(command, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if code := Run([]string{command, "--project-dir", root}, &stdout, &stderr); code != 2 {
				t.Fatalf("%s should be unknown: code=%d stdout=%q stderr=%q", command, code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), `unknown command "`+command+`"`) {
				t.Fatalf("expected unknown command message, got %q", stderr.String())
			}
		})
	}
}

func TestRunDiffCommand(t *testing.T) {
	root := writeCLIProject(t)
	store := state.Open(filepath.Join(root, ".fbt", "state"))
	writeCLIArtifactVersion(t, store, root, "artifact_version.knowledge_ops.case_summaries.sha256_1111111111111111111111111111111111111111111111111111111111111111", "target/artifacts/support/case_summaries_old", "old")
	newVersion := writeCLIArtifactVersion(t, store, root, "artifact_version.knowledge_ops.case_summaries.sha256_2222222222222222222222222222222222222222222222222222222222222222", "target/artifacts/support/case_summaries_new", "new")
	if err := store.WriteState(state.Snapshot{
		CurrentArtifacts: map[string]state.ArtifactPointer{
			newVersion.ArtifactID: {
				ArtifactID:       newVersion.ArtifactID,
				CurrentVersionID: newVersion.VersionID,
				LogicalPath:      newVersion.LogicalPath,
				Confidence:       "structural",
			},
		},
		LatestRuns: map[string]state.LatestRun{},
	}); err != nil {
		t.Fatal(err)
	}

	var diffOut bytes.Buffer
	var diffErr bytes.Buffer
	if code := Run([]string{"diff", "case_summaries", "--against", "previous", "--project-dir", root}, &diffOut, &diffErr); code != 0 {
		t.Fatalf("diff failed: code=%d stdout=%q stderr=%q", code, diffOut.String(), diffErr.String())
	}
	if !strings.Contains(diffOut.String(), "+new") || !strings.Contains(diffOut.String(), "changed: Summary") {
		t.Fatalf("unexpected diff output: %q", diffOut.String())
	}
}

func TestRunDoctorShowsMissingRunnerDiagnostic(t *testing.T) {
	root := writeCLIProject(t)
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
	repoRoot := repoRoot(t)
	command := filepath.Join(root, "bin", "fbt-openai-runner")
	writeFile(t, root, "bin/fbt-openai-runner", "#!/bin/sh\nexec go run "+shellQuote(filepath.Join(repoRoot, "runners", "fake"))+"\n")
	if err := os.Chmod(command, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "fs_project.yml", strings.ReplaceAll(readFile(t, filepath.Join(root, "fs_project.yml")), "command: fbt-openai-runner", "command: bin/fbt-openai-runner"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"doctor", "--project-dir", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("doctor failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "RUNNER_COMMAND_OK") {
		t.Fatalf("expected ok diagnostic, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "RUNNER_CAPABILITIES_OK") {
		t.Fatalf("expected capability diagnostic, got %q", stdout.String())
	}
}

func TestRunDoctorShowsMixedRunnerDiagnosticStatuses(t *testing.T) {
	root := writeCLIProject(t)
	command := filepath.Join(root, "bin", "fbt-openai-runner")
	writeFile(t, root, "bin/fbt-openai-runner", "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(command, 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.ReplaceAll(readFile(t, filepath.Join(root, "fs_project.yml")), "command: fbt-openai-runner", "command: bin/fbt-openai-runner\n    env: [\"OPENAI_API_KEY\"]")
	writeFile(t, root, "fs_project.yml", content)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"doctor", "--project-dir", root}, &stdout, &stderr); code != 6 {
		t.Fatalf("expected missing env exit code 6, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "error RUNNER_ENV_MISSING") {
		t.Fatalf("expected missing env diagnostic, got %q", out)
	}
	if !strings.Contains(out, "ok RUNNER_COMMAND_OK") {
		t.Fatalf("expected executable command to remain ok, got %q", out)
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
		VersionID:   "artifact_version.knowledge_ops.case_summaries.sha256_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ArtifactID:  "artifact.knowledge_ops.case_summaries",
		LogicalPath: "target/artifacts/support/case_summaries/",
		StoragePath: "target/artifacts/support/case_summaries/",
		GeneratedBy: "transform_run.run_1",
		Confidence:  "structural",
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

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
