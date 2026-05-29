package claudecodeadapter

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

func TestClaudeCodeAdapterProtocol(t *testing.T) {
	temp := t.TempDir()
	fixture := filepath.Join(temp, "claude-code-fixture.sh")
	argsPath := filepath.Join(temp, "args.txt")
	if err := os.WriteFile(fixture, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$FBT_CAPTURE_ARGS\"\nprintf '# Claude Code Output\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FBT_CLAUDE_CODE_COMMAND", fixture)
	t.Setenv("FBT_CAPTURE_ARGS", argsPath)
	response, workOutputs := runFixture(t)
	if response["result"].(map[string]any)["status"] != "success" {
		t.Fatalf("unexpected response: %+v", response)
	}
	args := readFile(t, argsPath)
	for _, want := range []string{
		"--permission-mode\ndontAsk\n",
		"--allowedTools\nRead\n",
		"--disallowedTools\nBash\n",
		"--max-budget-usd\n1.25\n",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("expected args to contain %q, got:\n%s", want, args)
		}
	}
	data, err := os.ReadFile(filepath.Join(workOutputs, "output"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Claude Code Output") {
		t.Fatalf("unexpected output: %s", string(data))
	}
}

func TestClaudeCodeAdapterFailsClosedWhenNetworkDenied(t *testing.T) {
	temp := t.TempDir()
	invoked := filepath.Join(temp, "invoked")
	fixture := filepath.Join(temp, "claude-code-fixture.sh")
	if err := os.WriteFile(fixture, []byte("#!/bin/sh\ntouch \"$FBT_INVOKED\"\nprintf '# should not run\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FBT_CLAUDE_CODE_COMMAND", fixture)
	t.Setenv("FBT_INVOKED", invoked)
	response, _ := runFixtureWithPolicy(t, map[string]any{"network": false})
	errObj, ok := response["error"].(map[string]any)
	if !ok || !strings.Contains(errObj["message"].(string), "policy denies network") {
		t.Fatalf("expected policy mapping error, got %+v", response)
	}
	if _, err := os.Stat(invoked); !os.IsNotExist(err) {
		t.Fatalf("external CLI should not be invoked, stat err=%v", err)
	}
}

func runFixture(t *testing.T) (map[string]any, string) {
	return runFixtureWithPolicy(t, map[string]any{
		"network": true,
		"tools": map[string]any{
			"allow": []any{"read_source"},
			"deny":  []any{"shell"},
		},
		"limits": map[string]any{"timeout_seconds": 30, "max_cost_usd": 1.25},
	})
}

func runFixtureWithPolicy(t *testing.T, policy map[string]any) (map[string]any, string) {
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
			"transform_run_id": "transform_run.claude",
			"transform":        map[string]any{"name": "agent_output", "type": "agent"},
			"inputs":           []any{map[string]any{"name": "source", "resolved_paths": []any{source}}},
			"assets":           []any{map[string]any{"name": "prompt", "absolute_path": asset}},
			"outputs":          []any{map[string]any{"name": "output", "artifact_type": "markdown"}},
			"model":            map[string]any{"provider": "conformance", "name": "fixture"},
			"policy":           policy,
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
	if len(lines) < 2 {
		t.Fatalf("expected protocol messages, got %d: %s", len(lines), output.String())
	}
	var response map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &response); err != nil {
		t.Fatal(err)
	}
	return response, workOutputs
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
