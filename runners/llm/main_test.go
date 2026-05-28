package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/internal/protocol"
)

func TestLLMRunnerProtocolReportsUsageAndOutput(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	client, err := protocol.Start(context.Background(), "go", []string{"run", "."}, protocol.Options{Dir: wd, Env: os.Environ()})
	if err != nil {
		t.Fatalf("start llm runner: %v", err)
	}
	defer client.Close()

	init, err := client.Initialize(context.Background(), protocol.InitializeParams{})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if init.Capabilities["usage_reporting"] != true {
		t.Fatalf("expected usage reporting capability: %+v", init.Capabilities)
	}

	work := filepath.Join(t.TempDir(), "work", "outputs")
	outcome, err := client.RunTransform(context.Background(), protocol.RunTransformParams{
		Mode:           "run",
		TransformRunID: "transform_run.llm",
		Transform:      map[string]any{"name": "case_summaries", "type": "llm"},
		Model:          map[string]any{"provider": "local", "name": "mock-gpt"},
		Outputs: []any{
			map[string]any{"name": "case_summaries", "artifact_type": "markdown_directory", "declared_path": "target/artifacts/case_summaries/"},
		},
		Work: map[string]any{"outputs": work},
	})
	if err != nil {
		t.Fatalf("run transform: %v", err)
	}
	if outcome.Result.Status != "success" {
		t.Fatalf("unexpected status: %s", outcome.Result.Status)
	}
	if outcome.Result.Usage["fbt.usage.total_tokens"] == nil {
		t.Fatalf("missing usage: %+v", outcome.Result.Usage)
	}
	if outcome.Result.Provenance["model"] != "mock-gpt" {
		t.Fatalf("missing provenance: %+v", outcome.Result.Provenance)
	}
	if len(outcome.Events) != 1 || outcome.Events[0].EventType != "usage" {
		t.Fatalf("expected usage event, got %+v", outcome.Events)
	}
	if len(outcome.OutputCandidates) != 1 {
		t.Fatalf("expected output candidate, got %d", len(outcome.OutputCandidates))
	}
	if _, err := os.Stat(filepath.Join(work, "case_summaries", "index.md")); err != nil {
		t.Fatalf("expected generated output: %v", err)
	}
}
