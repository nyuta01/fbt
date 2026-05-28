package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type runParams struct {
	Transform struct {
		Name string `json:"name"`
	} `json:"transform"`
	TransformRunID string `json:"transform_run_id"`
	Work           struct {
		Outputs string `json:"outputs"`
	} `json:"work"`
	Outputs []struct {
		Name         string `json:"name"`
		ArtifactType string `json:"artifact_type"`
		DeclaredPath string `json:"declared_path"`
	} `json:"outputs"`
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
				"runner": map[string]any{
					"name":     "fbt-fake-runner",
					"version":  "0.1.0",
					"language": "go",
				},
				"protocol": map[string]any{
					"version": "0.1",
					"framing": "jsonl",
				},
				"capabilities": map[string]any{
					"transform_types":   []string{"command", "llm", "agent", "template", "compose"},
					"artifact_types":    []string{"markdown", "markdown_directory", "text", "directory"},
					"stream_events":     true,
					"output_candidates": true,
					"supports_cancel":   true,
				},
			})
		case "initialized":
		case "fbt/runTransform":
			if err := runFake(req); err != nil {
				writeError(req.ID, -32099, err.Error())
			}
		case "$/cancelRequest":
			os.Exit(0)
		default:
			writeError(req.ID, -32601, "method not found")
		}
	}
}

func runFake(req request) error {
	var params runParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return err
	}
	if params.Work.Outputs == "" {
		return fmt.Errorf("work.outputs is required")
	}
	if len(params.Outputs) == 0 {
		params.Outputs = append(params.Outputs, struct {
			Name         string `json:"name"`
			ArtifactType string `json:"artifact_type"`
			DeclaredPath string `json:"declared_path"`
		}{Name: "output", ArtifactType: "markdown"})
	}
	if err := os.MkdirAll(params.Work.Outputs, 0o755); err != nil {
		return err
	}
	writeNotification("fbt/event", map[string]any{
		"request_id":       req.ID,
		"transform_run_id": params.TransformRunID,
		"event_type":       "progress",
		"level":            "info",
		"message":          "fake runner generating outputs",
	})

	var outputs []map[string]any
	for _, declared := range params.Outputs {
		name := declared.Name
		if name == "" {
			name = "output"
		}
		outPath := filepath.Join(params.Work.Outputs, name)
		if isDirectoryType(declared.ArtifactType) {
			if err := os.MkdirAll(outPath, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(outPath, "index.md"), []byte("# Fake Output\n"), 0o644); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(outPath, []byte("# Fake Output\n"), 0o644); err != nil {
				return err
			}
		}
		outputs = append(outputs, map[string]any{
			"name":          name,
			"artifact_type": declared.ArtifactType,
			"path":          outPath,
			"declared_path": declared.DeclaredPath,
		})
	}
	writeNotification("fbt/outputCandidate", map[string]any{
		"request_id":       req.ID,
		"transform_run_id": params.TransformRunID,
		"outputs":          outputs,
	})
	writeResponse(req.ID, map[string]any{
		"status":           "success",
		"transform_run_id": params.TransformRunID,
		"outputs":          outputs,
		"warnings":         []string{},
	})
	return nil
}

func isDirectoryType(artifactType string) bool {
	return artifactType == "directory" || strings.HasSuffix(artifactType, "_directory")
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
