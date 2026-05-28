package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/internal/protocol"
)

func TestCommandRunnerProtocol(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	client, err := protocol.Start(context.Background(), "go", []string{"run", "."}, protocol.Options{Dir: wd, Env: os.Environ()})
	if err != nil {
		t.Fatalf("start command runner: %v", err)
	}
	defer client.Close()

	if _, err := client.Initialize(context.Background(), protocol.InitializeParams{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	temp := t.TempDir()
	script := filepath.Join(temp, "write-output.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '# Command Output\\n' > \"$FBT_WORK_OUTPUTS/result.md\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	workRoot := filepath.Join(temp, "work")
	workOutputs := filepath.Join(workRoot, "outputs")
	outcome, err := client.RunTransform(context.Background(), protocol.RunTransformParams{
		Mode:           "run",
		TransformRunID: "transform_run.command",
		Transform:      map[string]any{"command": []string{script}},
		Work:           map[string]any{"root": workRoot, "temp": filepath.Join(workRoot, "tmp"), "outputs": workOutputs},
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if outcome.Result.Status != "success" {
		t.Fatalf("unexpected status: %s", outcome.Result.Status)
	}
	if _, err := os.Stat(filepath.Join(workOutputs, "result.md")); err != nil {
		t.Fatalf("expected command output: %v", err)
	}
	if len(outcome.OutputCandidates) != 1 {
		t.Fatalf("expected output candidate notification, got %d", len(outcome.OutputCandidates))
	}
}
