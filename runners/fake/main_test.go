package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/internal/protocol"
)

func TestFakeRunnerProtocol(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	client, err := protocol.Start(context.Background(), "go", []string{"run", "."}, protocol.Options{Dir: wd, Env: os.Environ()})
	if err != nil {
		t.Fatalf("start fake runner: %v", err)
	}
	defer client.Close()

	if _, err := client.Initialize(context.Background(), protocol.InitializeParams{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	work := filepath.Join(t.TempDir(), "work")
	outcome, err := client.RunTransform(context.Background(), protocol.RunTransformParams{
		Mode:           "run",
		TransformRunID: "transform_run.fake",
		Transform:      map[string]any{"name": "fake_transform"},
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
	if _, err := os.Stat(filepath.Join(work, "case_summaries", "index.md")); err != nil {
		t.Fatalf("expected fake output: %v", err)
	}
}
