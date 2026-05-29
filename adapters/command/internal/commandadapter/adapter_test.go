package commandadapter

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

func TestCommandAdapterProtocol(t *testing.T) {
	temp := t.TempDir()
	script := filepath.Join(temp, "write-output.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '# Command Output\\n' > \"$FBT_WORK_OUTPUTS/result.md\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	workRoot := filepath.Join(temp, "work")
	workOutputs := filepath.Join(workRoot, "outputs")
	input := strings.NewReader(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":"run","method":"fbt/runTransform","params":{"mode":"run","transform_run_id":"transform_run.command","transform":{"command":[` + quote(script) + `]},"work":{"root":` + quote(workRoot) + `,"temp":` + quote(filepath.Join(workRoot, "tmp")) + `,"outputs":` + quote(workOutputs) + `}}}` + "\n")
	var output bytes.Buffer
	if err := stdiojsonrpc.Serve(context.Background(), input, &output, Handler()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workOutputs, "result.md")); err != nil {
		t.Fatalf("expected command output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected init response, event, output candidate, and run response; got %d: %s", len(lines), output.String())
	}
	var runResponse map[string]any
	if err := json.Unmarshal([]byte(lines[3]), &runResponse); err != nil {
		t.Fatal(err)
	}
	result := runResponse["result"].(map[string]any)
	if result["status"] != "success" {
		t.Fatalf("unexpected response: %+v", runResponse)
	}
}

func quote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
