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
	argsPath := filepath.Join(temp, "args.txt")
	if err := os.WriteFile(fixture, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$FBT_CAPTURE_ARGS\"\nout=''\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = '--output-last-message' ]; then shift; out=\"$1\"; fi\n  shift || true\ndone\nif [ -n \"$out\" ]; then printf '# Codex Output\\n' > \"$out\"; fi\nprintf '# ignored stdout\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FBT_CODEX_CLI_COMMAND", fixture)
	t.Setenv("FBT_CAPTURE_ARGS", argsPath)
	response, workOutputs := runFixture(t)
	if response["result"].(map[string]any)["status"] != "success" {
		t.Fatalf("unexpected response: %+v", response)
	}
	args := readFile(t, argsPath)
	if !strings.Contains(args, "--sandbox\nread-only\n") {
		t.Fatalf("expected read-only sandbox args, got:\n%s", args)
	}
	data, err := os.ReadFile(filepath.Join(workOutputs, "output"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Codex Output") {
		t.Fatalf("unexpected output: %s", string(data))
	}
}

func TestCodexCLIAdapterFailsClosedWhenNetworkDenied(t *testing.T) {
	temp := t.TempDir()
	invoked := filepath.Join(temp, "invoked")
	fixture := filepath.Join(temp, "codex-cli-fixture.sh")
	if err := os.WriteFile(fixture, []byte("#!/bin/sh\ntouch \"$FBT_INVOKED\"\nprintf '# should not run\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FBT_CODEX_CLI_COMMAND", fixture)
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

func TestCodexCLIAdapterFailsBeforeCLIWhenStagedFileExceedsLimit(t *testing.T) {
	for _, tc := range []struct {
		name        string
		sourceBytes []byte
		assetBytes  []byte
		wantPath    string
	}{
		{name: "source", sourceBytes: largeFixtureContent(), wantPath: "source.md"},
		{name: "asset", assetBytes: largeFixtureContent(), wantPath: "prompt.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			temp := t.TempDir()
			invoked := filepath.Join(temp, "invoked")
			fixture := filepath.Join(temp, "codex-cli-fixture.sh")
			if err := os.WriteFile(fixture, []byte("#!/bin/sh\ntouch \"$FBT_INVOKED\"\nprintf '# should not run\\n'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("FBT_CODEX_CLI_COMMAND", fixture)
			t.Setenv("FBT_INVOKED", invoked)

			response, _ := runFixtureWithOptions(t, fixtureOptions{
				policy:      map[string]any{"network": true},
				sourceBytes: tc.sourceBytes,
				assetBytes:  tc.assetBytes,
			})
			errObj, ok := response["error"].(map[string]any)
			if !ok {
				t.Fatalf("expected staging error, got %+v", response)
			}
			message := errObj["message"].(string)
			if !strings.Contains(message, "staging limit") || !strings.Contains(message, tc.wantPath) {
				t.Fatalf("expected actionable staging error, got %q", message)
			}
			if _, err := os.Stat(invoked); !os.IsNotExist(err) {
				t.Fatalf("external CLI should not be invoked, stat err=%v", err)
			}
		})
	}
}

func runFixture(t *testing.T) (map[string]any, string) {
	return runFixtureWithPolicy(t, map[string]any{
		"network": true,
		"limits":  map[string]any{"timeout_seconds": 30},
	})
}

func runFixtureWithPolicy(t *testing.T, policy map[string]any) (map[string]any, string) {
	return runFixtureWithOptions(t, fixtureOptions{policy: policy})
}

type fixtureOptions struct {
	policy      map[string]any
	sourceBytes []byte
	assetBytes  []byte
}

func runFixtureWithOptions(t *testing.T, options fixtureOptions) (map[string]any, string) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source.md")
	asset := filepath.Join(root, "prompt.md")
	sourceBytes := options.sourceBytes
	if sourceBytes == nil {
		sourceBytes = []byte("# Source\n")
	}
	assetBytes := options.assetBytes
	if assetBytes == nil {
		assetBytes = []byte("# Prompt\n")
	}
	if err := os.WriteFile(source, sourceBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(asset, assetBytes, 0o644); err != nil {
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
			"model":            map[string]any{"provider": "conformance", "name": "fixture"},
			"policy":           options.policy,
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

func largeFixtureContent() []byte {
	return bytes.Repeat([]byte("x"), stagedInputMaxBytes+1)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
