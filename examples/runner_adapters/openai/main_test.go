package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nyuta01/fbt/internal/protocol"
)

func TestOpenAIRunnerProtocolCallsResponsesAPI(t *testing.T) {
	var sawAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		sawAuth = r.Header.Get("Authorization") == "Bearer test-key"
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-test" || !strings.Contains(body["input"].(string), "Incident Response Runbook Format") {
			t.Fatalf("unexpected request body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"resp_test",
			"output":[{"type":"message","content":[{"type":"output_text","text":"# Generated Manual\n\n## Purpose\n\nTest output.\n"}]}],
			"usage":{"input_tokens":123,"output_tokens":45,"total_tokens":168}
		}`))
	}))
	defer server.Close()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	client, err := protocol.Start(context.Background(), "go", []string{"run", "."}, protocol.Options{
		Dir: wd,
		Env: append(os.Environ(), "OPENAI_API_KEY=test-key", "OPENAI_RESPONSES_ENDPOINT="+server.URL+"/v1/responses"),
	})
	if err != nil {
		t.Fatalf("start openai runner: %v", err)
	}
	defer client.Close()

	init, err := client.Initialize(context.Background(), protocol.InitializeParams{})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if init.Capabilities["usage_reporting"] != true {
		t.Fatalf("expected usage reporting capability: %+v", init.Capabilities)
	}

	root := t.TempDir()
	assetPath := filepath.Join(root, "format.md")
	inputPath := filepath.Join(root, "incident.jsonl")
	if err := os.WriteFile(assetPath, []byte("# Incident Response Runbook Format\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, []byte(`{"incident_id":"INC-1","event":"latency"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, "work", "outputs")
	outcome, err := client.RunTransform(context.Background(), protocol.RunTransformParams{
		Mode:           "run",
		TransformRunID: "transform_run.openai",
		Transform:      map[string]any{"name": "incident_response_runbook", "type": "llm"},
		Model:          map[string]any{"provider": "openai", "name": "gpt-test"},
		Inputs: []any{
			map[string]any{"kind": "source", "name": "event_logs", "resolved_paths": []any{inputPath}},
		},
		Assets: []any{
			map[string]any{"name": "incident_runbook_format", "absolute_path": assetPath},
		},
		Outputs: []any{
			map[string]any{"name": "incident_response_runbook", "artifact_type": "markdown", "declared_path": "target/artifacts/runbooks/incident_response_runbook.md"},
		},
		Work: map[string]any{"outputs": work},
	})
	if err != nil {
		t.Fatalf("run transform: %v", err)
	}
	if !sawAuth {
		t.Fatal("expected bearer auth header")
	}
	if outcome.Result.Status != "success" {
		t.Fatalf("unexpected status: %s", outcome.Result.Status)
	}
	if outcome.Result.Usage["fbt.usage.total_tokens"] != float64(168) {
		t.Fatalf("missing usage: %+v", outcome.Result.Usage)
	}
	if outcome.Result.Provenance["response_id"] != "resp_test" {
		t.Fatalf("missing provenance: %+v", outcome.Result.Provenance)
	}
	if len(outcome.OutputCandidates) != 1 {
		t.Fatalf("expected output candidate, got %d", len(outcome.OutputCandidates))
	}
	data, err := os.ReadFile(filepath.Join(work, "incident_response_runbook"))
	if err != nil {
		t.Fatalf("expected generated output: %v", err)
	}
	if !strings.Contains(string(data), "Generated Manual") {
		t.Fatalf("unexpected output: %s", string(data))
	}
}

func TestProjectRootFromWorkResolvesRelativeInputs(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "fbt-project")
	workOutputs := filepath.Join(root, ".fbt", "work", "run-1", "outputs")
	if got := projectRootFromWork(workOutputs); got != root {
		t.Fatalf("project root = %q, want %q", got, root)
	}
	if got := resolvePath(root, "data/input.jsonl"); got != filepath.Join(root, "data", "input.jsonl") {
		t.Fatalf("resolved path = %q", got)
	}
}
