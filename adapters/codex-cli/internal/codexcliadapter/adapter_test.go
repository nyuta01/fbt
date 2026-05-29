package codexcliadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nyuta01/fbt/sdk/go/stdiojsonrpc"
)

func TestCodexCLIAdapterProtocol(t *testing.T) {
	temp := t.TempDir()
	fixture := filepath.Join(temp, "codex-cli-fixture.sh")
	if err := os.WriteFile(fixture, []byte("#!/bin/sh\nout=''\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = '--output-last-message' ]; then shift; out=\"$1\"; fi\n  shift || true\ndone\nif [ -n \"$out\" ]; then printf '# Codex Output\\n' > \"$out\"; fi\nprintf '# ignored stdout\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FBT_CODEX_CLI_COMMAND", fixture)
	response, workOutputs := runFixture(t)
	if response["result"].(map[string]any)["status"] != "success" {
		t.Fatalf("unexpected response: %+v", response)
	}
	data, err := os.ReadFile(filepath.Join(workOutputs, "output"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Codex Output") {
		t.Fatalf("unexpected output: %s", string(data))
	}
}

func runFixture(t *testing.T) (map[string]any, string) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source.md")
	asset := filepath.Join(root, "prompt.md")
	if err := os.WriteFile(source, []byte("# Source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(asset, []byte("# Prompt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workRoot := filepath.Join(root, ".fbt", "work", "test")
	workOutputs := filepath.Join(workRoot, "outputs")
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      "run",
		"method":  "fbt/runTransform",
		"params": map[string]any{
			"mode":             "run",
			"transform_run_id": "transform_run.codex",
			"transform":        map[string]any{"name": "agent_output", "type": "agent"},
			"inputs":           []any{map[string]any{"name": "source", "resolved_paths": []any{source}}},
			"assets":           []any{map[string]any{"name": "prompt", "absolute_path": asset}},
			"outputs":          []any{map[string]any{"name": "output", "artifact_type": "markdown"}},
			"work":             map[string]any{"root": workRoot, "temp": filepath.Join(workRoot, "tmp"), "outputs": workOutputs},
		},
	}
	var input bytes.Buffer
	input.WriteString(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}` + "\n")
	if err := json.NewEncoder(&input).Encode(request); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := stdiojsonrpc.Serve(context.Background(), &input, &output, Handler()); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 protocol messages, got %d: %s", len(lines), output.String())
	}
	var response map[string]any
	if err := json.Unmarshal([]byte(lines[3]), &response); err != nil {
		t.Fatal(err)
	}
	return response, workOutputs
}
