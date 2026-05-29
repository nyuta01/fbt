package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/internal/protocol"
)

func TestAgentRunnerProtocolReportsToolCallsUsageAndOutput(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	client, err := protocol.Start(context.Background(), "go", []string{"run", "."}, protocol.Options{Dir: wd, Env: os.Environ()})
	if err != nil {
		t.Fatalf("start agent runner: %v", err)
	}
	defer client.Close()

	init, err := client.Initialize(context.Background(), protocol.InitializeParams{})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if init.Capabilities["tool_call_log"] != true {
		t.Fatalf("expected tool call capability: %+v", init.Capabilities)
	}

	work := filepath.Join(t.TempDir(), "work", "outputs")
	outcome, err := client.RunTransform(context.Background(), protocol.RunTransformParams{
		Mode:           "run",
		TransformRunID: "transform_run.agent",
		Transform:      map[string]any{"name": "weekly_support_insights", "type": "agent"},
		Model:          map[string]any{"provider": "local", "name": "mock-agent"},
		Tools: []any{
			map[string]any{"name": "read_artifact"},
			map[string]any{"name": "write_artifact"},
		},
		Outputs: []any{
			map[string]any{"name": "weekly_support_insights", "artifact_type": "markdown", "declared_path": "target/artifacts/weekly_support_insights.md"},
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
	toolEvents := 0
	for _, event := range outcome.Events {
		if event.EventType == "tool_call.completed" {
			toolEvents++
			if event.ToolCall["arguments_redacted"] == nil {
				t.Fatalf("tool call was not redacted: %+v", event.ToolCall)
			}
		}
	}
	if toolEvents != 2 {
		t.Fatalf("expected two tool-call events, got %+v", outcome.Events)
	}
	if len(outcome.OutputCandidates) != 1 {
		t.Fatalf("expected output candidate, got %d", len(outcome.OutputCandidates))
	}
	if _, err := os.Stat(filepath.Join(work, "weekly_support_insights")); err != nil {
		t.Fatalf("expected generated output: %v", err)
	}
}
