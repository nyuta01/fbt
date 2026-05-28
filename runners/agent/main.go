package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type runParams struct {
	Mode           string         `json:"mode"`
	TransformRunID string         `json:"transform_run_id"`
	Transform      transformInfo  `json:"transform"`
	Model          map[string]any `json:"model"`
	Tools          []any          `json:"tools"`
	Outputs        []declaredOut  `json:"outputs"`
	Work           workInfo       `json:"work"`
}

type transformInfo struct {
	UniqueID string `json:"unique_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

type declaredOut struct {
	Name         string `json:"name"`
	ArtifactType string `json:"artifact_type"`
	DeclaredPath string `json:"declared_path"`
}

type workInfo struct {
	Outputs string `json:"outputs"`
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
			writeResponse(req.ID, initializeResult())
		case "initialized":
		case "fbt/runTransform":
			if err := runAgent(req); err != nil {
				writeError(req.ID, -32099, err.Error())
			}
		case "$/cancelRequest":
			os.Exit(0)
		default:
			writeError(req.ID, -32601, "method not found")
		}
	}
}

func initializeResult() map[string]any {
	return map[string]any{
		"runner": map[string]any{
			"name":     "fbt-demo-agent-runner",
			"version":  "0.1.0",
			"language": "go",
		},
		"protocol": map[string]any{
			"version": "0.1",
			"framing": "jsonl",
		},
		"capabilities": map[string]any{
			"transform_types":   []string{"agent"},
			"artifact_types":    []string{"markdown", "markdown_directory", "text", "directory"},
			"stream_events":     true,
			"tool_call_log":     true,
			"usage_reporting":   true,
			"cost_estimation":   true,
			"output_candidates": true,
			"supports_dry_run":  true,
			"supports_cancel":   true,
		},
	}
}

func runAgent(req request) error {
	var params runParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return err
	}
	if params.Transform.Name == "" {
		params.Transform.Name = "agent_transform"
	}
	outputs := params.Outputs
	if len(outputs) == 0 {
		outputs = []declaredOut{{Name: "output", ArtifactType: "markdown"}}
	}
	tools := toolNames(params.Tools)
	usage := usageFor(params.Transform.Name, len(tools), len(outputs))
	provenance := provenanceFor(params.Model, tools)

	for i, tool := range tools {
		toolID := fmt.Sprintf("tool_%03d", i+1)
		writeNotification("fbt/event", map[string]any{
			"request_id":       req.ID,
			"transform_run_id": params.TransformRunID,
			"time":             time.Now().UTC().Format(time.RFC3339),
			"event_type":       "tool_call.completed",
			"level":            "info",
			"message":          "demo agent tool call completed",
			"attributes": map[string]any{
				"gen_ai.tool.call.id": toolID,
				"gen_ai.tool.name":    tool,
				"gen_ai.tool.type":    "function",
				"fbt.tool.status":     "success",
			},
			"tool_call": map[string]any{
				"id":                 toolID,
				"name":               tool,
				"arguments_redacted": map[string]any{"transform": params.Transform.Name},
				"status":             "success",
			},
		})
	}
	writeNotification("fbt/event", map[string]any{
		"request_id":       req.ID,
		"transform_run_id": params.TransformRunID,
		"time":             time.Now().UTC().Format(time.RFC3339),
		"event_type":       "usage",
		"level":            "info",
		"message":          "demo agent runner completed deterministic plan",
		"attributes": map[string]any{
			"gen_ai.usage.input_tokens":   usage["gen_ai.usage.input_tokens"],
			"gen_ai.usage.output_tokens":  usage["gen_ai.usage.output_tokens"],
			"fbt.usage.total_tokens":      usage["fbt.usage.total_tokens"],
			"fbt.estimated_cost_usd":      usage["fbt.estimated_cost_usd"],
			"fbt.runner.demo":             true,
			"fbt.runner.tool_call_events": true,
		},
	})

	if params.Mode == "plan" || params.Mode == "dry_run" {
		writeResponse(req.ID, map[string]any{
			"status":           "success",
			"transform_run_id": params.TransformRunID,
			"usage":            usage,
			"provenance":       provenance,
			"warnings":         []string{"dry run only; no output candidates written"},
		})
		return nil
	}
	if params.Work.Outputs == "" {
		return fmt.Errorf("work.outputs is required")
	}
	if err := os.MkdirAll(params.Work.Outputs, 0o755); err != nil {
		return err
	}
	content := agentContent(params.Transform, tools)
	var candidates []map[string]any
	for _, output := range outputs {
		candidate, err := writeOutput(params.Work.Outputs, output, content)
		if err != nil {
			return err
		}
		candidates = append(candidates, candidate)
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
		"usage":            usage,
		"provenance":       provenance,
		"warnings":         []string{},
	})
	return nil
}

func agentContent(transform transformInfo, tools []string) string {
	var builder strings.Builder
	builder.WriteString("# Agent Run: ")
	builder.WriteString(title(transform.Name))
	builder.WriteString("\n\n## Plan\n\n- inspect inputs\n- apply declared tools\n- write candidate artifact\n\n## Tool Calls\n\n")
	for _, tool := range tools {
		builder.WriteString("- ")
		builder.WriteString(tool)
		builder.WriteString(": success\n")
	}
	builder.WriteString("\n## Result\n\nGenerated by the deterministic FBT demo agent runner.\n")
	return builder.String()
}

func writeOutput(root string, output declaredOut, content string) (map[string]any, error) {
	name := output.Name
	if name == "" {
		name = "output"
	}
	artifactType := output.ArtifactType
	if artifactType == "" {
		artifactType = "markdown"
	}
	path := filepath.Join(root, name)
	if isDirectoryType(artifactType) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(path, "index.md"), []byte(content), 0o644); err != nil {
			return nil, err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"name":          name,
		"artifact_type": artifactType,
		"path":          path,
		"declared_path": output.DeclaredPath,
	}, nil
}

func toolNames(raw []any) []string {
	var names []string
	for _, item := range raw {
		if text, ok := item.(string); ok && text != "" {
			names = append(names, text)
			continue
		}
		if object, ok := item.(map[string]any); ok {
			if name, ok := object["name"].(string); ok && name != "" {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		return []string{"read_artifact", "write_artifact"}
	}
	return names
}

func provenanceFor(model map[string]any, tools []string) map[string]any {
	return map[string]any{
		"runner":                "fbt-demo-agent-runner",
		"runner_version":        "0.1.0",
		"model_provider":        stringValue(model, "provider", "demo"),
		"model":                 stringValue(model, "name", "deterministic-demo-agent"),
		"model_parameters_hash": hashJSON(model),
		"tools":                 tools,
		"materials":             []any{},
	}
}

func usageFor(transformName string, tools, outputs int) map[string]any {
	inputTokens := 160 + len(transformName) + tools*25
	outputTokens := 110 + outputs*35
	total := inputTokens + outputTokens
	return map[string]any{
		"gen_ai.usage.input_tokens":  inputTokens,
		"gen_ai.usage.output_tokens": outputTokens,
		"fbt.usage.total_tokens":     total,
		"fbt.estimated_cost_usd":     float64(total) * 0.000001,
	}
}

func stringValue(values map[string]any, key, fallback string) string {
	if value, ok := values[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

func hashJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "sha256:"
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func title(value string) string {
	parts := strings.Fields(strings.ReplaceAll(value, "_", " "))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
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
