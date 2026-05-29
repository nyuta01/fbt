package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClientInitializeAndRunTransform(t *testing.T) {
	client := startFakeRunner(t, "success")
	defer client.Close()

	initResult, err := client.Initialize(context.Background(), InitializeParams{
		Core: map[string]string{"name": "fbt-core", "version": "test"},
		Protocol: map[string]any{
			"versions": []string{"0.1"},
			"framing":  "jsonl",
		},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if initResult.Protocol["version"] != "0.1" {
		t.Fatalf("unexpected protocol result: %+v", initResult.Protocol)
	}

	outcome, err := client.RunTransform(context.Background(), RunTransformParams{
		Mode:           "run",
		InvocationID:   "inv_1",
		TransformRunID: "transform_run.run_1",
		Transform:      map[string]any{"name": "case_summaries"},
	})
	if err != nil {
		t.Fatalf("run transform: %v", err)
	}
	if outcome.Result.Status != "success" {
		t.Fatalf("unexpected status: %s", outcome.Result.Status)
	}
	if len(outcome.Events) != 1 {
		t.Fatalf("expected one event, got %d", len(outcome.Events))
	}
	if len(outcome.OutputCandidates) != 1 {
		t.Fatalf("expected one output candidate, got %d", len(outcome.OutputCandidates))
	}
}

func TestClientReturnsJSONRPCError(t *testing.T) {
	client := startFakeRunner(t, "error")
	defer client.Close()
	if _, err := client.Initialize(context.Background(), InitializeParams{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	_, err := client.RunTransform(context.Background(), RunTransformParams{Mode: "run"})
	if err == nil {
		t.Fatal("expected run error")
	}
	rpcErr, ok := err.(JSONRPCError)
	if !ok {
		t.Fatalf("expected JSONRPCError, got %T", err)
	}
	if rpcErr.Code != -32010 {
		t.Fatalf("unexpected error code: %d", rpcErr.Code)
	}
}

func TestClientReadsLargeRunnerMessages(t *testing.T) {
	client := startFakeRunner(t, "large_message")
	defer client.Close()
	if _, err := client.Initialize(context.Background(), InitializeParams{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	outcome, err := client.RunTransform(context.Background(), RunTransformParams{Mode: "run"})
	if err != nil {
		t.Fatalf("run transform: %v", err)
	}
	if len(outcome.Events) != 1 {
		t.Fatalf("expected one event, got %d", len(outcome.Events))
	}
	blob, ok := outcome.Events[0].Attributes["large_message"].(string)
	if !ok {
		t.Fatalf("expected large_message attribute, got %+v", outcome.Events[0].Attributes)
	}
	if len(blob) <= bufio.MaxScanTokenSize {
		t.Fatalf("expected message larger than scanner default, got %d bytes", len(blob))
	}
	if outcome.Result.Status != "success" {
		t.Fatalf("unexpected status: %s", outcome.Result.Status)
	}
}

func TestClientCancelsOnContext(t *testing.T) {
	client := startFakeRunner(t, "hang")
	defer client.Close()
	if _, err := client.Initialize(context.Background(), InitializeParams{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := client.RunTransform(ctx, RunTransformParams{Mode: "run"})
	if err == nil {
		t.Fatal("expected context timeout")
	}
}

func TestStartUsesArgsEnvAndDir(t *testing.T) {
	root := t.TempDir()
	client, err := Start(context.Background(), os.Args[0], []string{"-test.run=TestInvocationRunnerProcess", "--", "runner-arg"}, Options{
		Dir: root,
		Env: []string{
			"FBT_INVOCATION_RUNNER=1",
			"FBT_EXPECTED_ENV=present",
			"PATH=" + os.Getenv("PATH"),
		},
	})
	if err != nil {
		t.Fatalf("start invocation runner: %v", err)
	}
	defer client.Close()
	result, err := client.Initialize(context.Background(), InitializeParams{})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	expectedCWD, err := filepath.EvalSymlinks(root)
	if err != nil {
		expectedCWD = filepath.Clean(root)
	}
	if result.Runner["cwd"] != filepath.Clean(expectedCWD) {
		t.Fatalf("expected cwd %s, got %+v", expectedCWD, result.Runner)
	}
	if result.Runner["env"] != "present" {
		t.Fatalf("expected env passthrough, got %+v", result.Runner)
	}
	args, ok := result.Runner["args"].([]any)
	if !ok || len(args) == 0 || args[len(args)-1] != "runner-arg" {
		t.Fatalf("expected runner args, got %+v", result.Runner)
	}
}

func startFakeRunner(t *testing.T, mode string) *Client {
	t.Helper()
	ctx := context.Background()
	client, err := Start(ctx, os.Args[0], []string{"-test.run=TestFakeRunnerProcess"}, Options{
		Env: append(os.Environ(), "FBT_FAKE_RUNNER=1", "FBT_FAKE_RUNNER_MODE="+mode),
	})
	if err != nil {
		t.Fatalf("start fake runner: %v", err)
	}
	return client
}

func TestInvocationRunnerProcess(t *testing.T) {
	if os.Getenv("FBT_INVOCATION_RUNNER") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			os.Exit(2)
		}
		switch request.Method {
		case "initialize":
			cwd, _ := os.Getwd()
			args := make([]string, len(os.Args))
			copy(args, os.Args)
			writeFake(map[string]any{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]any{
					"runner": map[string]any{
						"name": "invocation",
						"cwd":  filepath.Clean(cwd),
						"env":  os.Getenv("FBT_EXPECTED_ENV"),
						"args": args,
					},
					"protocol":     map[string]any{"version": "0.1", "framing": "jsonl"},
					"capabilities": map[string]any{"run_transform": true},
				},
			})
		case "initialized":
		default:
			os.Exit(0)
		}
	}
	os.Exit(0)
}

func TestFakeRunnerProcess(t *testing.T) {
	if os.Getenv("FBT_FAKE_RUNNER") != "1" {
		return
	}
	mode := os.Getenv("FBT_FAKE_RUNNER_MODE")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			os.Exit(2)
		}
		switch request.Method {
		case "initialize":
			writeFake(map[string]any{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]any{
					"runner": map[string]any{"name": "fake"},
					"protocol": map[string]any{
						"version": "0.1",
						"framing": "jsonl",
					},
					"capabilities": map[string]any{"run_transform": true},
				},
			})
		case "initialized":
		case "fbt/runTransform":
			switch mode {
			case "error":
				writeFake(map[string]any{
					"jsonrpc": "2.0",
					"id":      request.ID,
					"error": map[string]any{
						"code":    -32010,
						"message": "Policy denied",
						"data":    map[string]any{"fbt_error_code": "POLICY_DENIED"},
					},
				})
			case "large_message":
				large := strings.Repeat("x", bufio.MaxScanTokenSize+1024)
				writeFake(map[string]any{
					"jsonrpc": "2.0",
					"method":  "fbt/event",
					"params": map[string]any{
						"request_id":       request.ID,
						"transform_run_id": "transform_run.run_1",
						"event_type":       "progress",
						"message":          "large structured notification",
						"attributes": map[string]any{
							"large_message": large,
						},
					},
				})
				writeFake(map[string]any{
					"jsonrpc": "2.0",
					"id":      request.ID,
					"result": map[string]any{
						"status":           "success",
						"transform_run_id": "transform_run.run_1",
					},
				})
			case "hang":
				select {}
			default:
				writeFake(map[string]any{
					"jsonrpc": "2.0",
					"method":  "fbt/event",
					"params": map[string]any{
						"request_id":       request.ID,
						"transform_run_id": "transform_run.run_1",
						"event_type":       "progress",
						"message":          "started",
					},
				})
				writeFake(map[string]any{
					"jsonrpc": "2.0",
					"method":  "fbt/outputCandidate",
					"params": map[string]any{
						"request_id":       request.ID,
						"transform_run_id": "transform_run.run_1",
						"outputs": []any{
							map[string]any{"name": "case_summaries", "path": "/tmp/out"},
						},
					},
				})
				writeFake(map[string]any{
					"jsonrpc": "2.0",
					"id":      request.ID,
					"result": map[string]any{
						"status":           "success",
						"transform_run_id": "transform_run.run_1",
						"outputs": []any{
							map[string]any{"name": "case_summaries"},
						},
					},
				})
			}
		case "$/cancelRequest":
			if strings.Contains(mode, "hang") {
				os.Exit(0)
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown method %s\n", request.Method)
			os.Exit(2)
		}
	}
	os.Exit(0)
}

func writeFake(value any) {
	data, err := json.Marshal(value)
	if err != nil {
		os.Exit(2)
	}
	fmt.Printf("%s\n", data)
}
