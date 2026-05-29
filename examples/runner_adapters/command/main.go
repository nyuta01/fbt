package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type runParams struct {
	Transform struct {
		Command []string `json:"command"`
	} `json:"transform"`
	TransformRunID string `json:"transform_run_id"`
	Work           struct {
		Root    string `json:"root"`
		Temp    string `json:"temp"`
		Outputs string `json:"outputs"`
	} `json:"work"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			writeError("", -32700, err.Error())
			continue
		}
		switch req.Method {
		case "initialize":
			writeResponse(req.ID, map[string]any{
				"runner": map[string]any{"name": "fbt-command-runner", "version": "0.1.0", "language": "go"},
				"protocol": map[string]any{
					"version": "0.1",
					"framing": "jsonl",
				},
				"capabilities": map[string]any{
					"transform_types":   []string{"command"},
					"artifact_types":    []string{"text", "markdown", "markdown_directory", "directory", "pdf"},
					"output_candidates": true,
					"supports_cancel":   true,
				},
			})
		case "initialized":
		case "fbt/runTransform":
			if err := runCommand(req); err != nil {
				writeError(req.ID, -32099, err.Error())
			}
		case "$/cancelRequest":
			os.Exit(0)
		default:
			writeError(req.ID, -32601, "method not found")
		}
	}
}

func runCommand(req request) error {
	var params runParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return err
	}
	if len(params.Transform.Command) == 0 {
		return fmt.Errorf("transform.command is required")
	}
	if params.Work.Outputs == "" {
		return fmt.Errorf("work.outputs is required")
	}
	if err := os.MkdirAll(params.Work.Outputs, 0o755); err != nil {
		return err
	}
	command := exec.Command(params.Transform.Command[0], params.Transform.Command[1:]...)
	if dir := os.Getenv("FBT_COMMAND_WORKDIR"); dir != "" {
		command.Dir = dir
	}
	command.Env = append(os.Environ(),
		"FBT_WORK_ROOT="+params.Work.Root,
		"FBT_WORK_TEMP="+params.Work.Temp,
		"FBT_WORK_OUTPUTS="+params.Work.Outputs,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w: %s", err, string(output))
	}
	candidates, err := outputCandidates(params.Work.Outputs)
	if err != nil {
		return err
	}
	writeNotification("fbt/outputCandidate", map[string]any{
		"request_id":       req.ID,
		"transform_run_id": params.TransformRunID,
		"outputs":          candidates,
	})
	writeResponse(req.ID, map[string]any{
		"status":           "success",
		"transform_run_id": params.TransformRunID,
		"outputs":          candidates,
		"warnings":         []string{},
	})
	return nil
}

func outputCandidates(root string) ([]map[string]any, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var candidates []map[string]any
	for _, entry := range entries {
		candidates = append(candidates, map[string]any{
			"name": entry.Name(),
			"path": filepath.Join(root, entry.Name()),
		})
	}
	return candidates, nil
}

func writeResponse(id string, result any) {
	write(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func writeNotification(method string, params any) {
	write(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func writeError(id string, code int, message string) {
	write(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}})
}

func write(value any) {
	data, err := json.Marshal(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal response: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", data)
}
