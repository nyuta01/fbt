package openaiadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nyuta01/fbt/sdk/go/stdiojsonrpc"
)

func TestOpenAIAdapterProtocolCallsResponsesAPI(t *testing.T) {
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
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_RESPONSES_ENDPOINT", server.URL+"/v1/responses")

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
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      "run",
		"method":  "fbt/runTransform",
		"params": map[string]any{
			"mode":             "run",
			"transform_run_id": "transform_run.openai",
			"transform":        map[string]any{"name": "incident_response_runbook", "type": "llm"},
			"model":            map[string]any{"provider": "openai", "name": "gpt-test"},
			"inputs": []any{
				map[string]any{"kind": "source", "name": "event_logs", "resolved_paths": []any{inputPath}},
			},
			"assets": []any{
				map[string]any{"name": "incident_runbook_format", "absolute_path": assetPath},
			},
			"outputs": []any{
				map[string]any{"name": "incident_response_runbook", "artifact_type": "markdown", "declared_path": "target/artifacts/runbooks/incident_response_runbook.md"},
			},
			"work": map[string]any{"outputs": work},
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
	if !sawAuth {
		t.Fatal("expected bearer auth header")
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
	if result["usage"].(map[string]any)["fbt.usage.total_tokens"] != float64(168) {
		t.Fatalf("missing usage: %+v", result["usage"])
	}
	if result["provenance"].(map[string]any)["response_id"] != "resp_test" {
		t.Fatalf("missing provenance: %+v", result["provenance"])
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
