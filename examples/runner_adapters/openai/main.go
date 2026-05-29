package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultEndpoint = "https://api.openai.com/v1/responses"
	maxContextBytes = 180_000
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type runParams struct {
	Mode           string           `json:"mode"`
	TransformRunID string           `json:"transform_run_id"`
	Transform      transformInfo    `json:"transform"`
	Inputs         []map[string]any `json:"inputs"`
	Outputs        []declaredOut    `json:"outputs"`
	Assets         []map[string]any `json:"assets"`
	Model          map[string]any   `json:"model"`
	Policy         map[string]any   `json:"policy"`
	Work           workInfo         `json:"work"`
}

type transformInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type declaredOut struct {
	Name         string `json:"name"`
	ArtifactType string `json:"artifact_type"`
	DeclaredPath string `json:"declared_path"`
}

type workInfo struct {
	Outputs string `json:"outputs"`
}

type responsesResult struct {
	ID     string         `json:"id"`
	Output []responseItem `json:"output"`
	Usage  map[string]any `json:"usage"`
	Error  *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

type responseItem struct {
	Type    string          `json:"type"`
	Content []responseBlock `json:"content"`
}

type responseBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
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
			if err := runOpenAI(req); err != nil {
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
			"name":     "fbt-runner-openai",
			"version":  "0.1.0",
			"language": "go",
		},
		"protocol": map[string]any{
			"version": "0.1",
			"framing": "jsonl",
		},
		"capabilities": map[string]any{
			"transform_types":   []string{"llm"},
			"artifact_types":    []string{"markdown", "markdown_directory", "text", "directory"},
			"stream_events":     true,
			"tool_call_log":     false,
			"usage_reporting":   true,
			"cost_estimation":   false,
			"output_candidates": true,
			"supports_dry_run":  true,
			"supports_cancel":   true,
		},
	}
}

func runOpenAI(req request) error {
	var params runParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return err
	}
	if params.Transform.Name == "" {
		params.Transform.Name = "openai_transform"
	}
	outputs := params.Outputs
	if len(outputs) == 0 {
		outputs = []declaredOut{{Name: "output", ArtifactType: "markdown"}}
	}
	model := stringValue(params.Model, "name", "gpt-5")
	provider := stringValue(params.Model, "provider", "openai")
	if params.Mode == "plan" || params.Mode == "dry_run" {
		writeResponse(req.ID, map[string]any{
			"status":           "success",
			"transform_run_id": params.TransformRunID,
			"provenance":       provenanceFor(model, provider, "", params.Model),
			"warnings":         []string{"dry run only; no OpenAI request sent"},
		})
		return nil
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return errors.New("OPENAI_API_KEY is required")
	}
	if params.Work.Outputs == "" {
		return errors.New("work.outputs is required")
	}
	prompt, err := buildPrompt(params)
	if err != nil {
		return err
	}
	start := time.Now()
	result, err := callResponses(apiKey, model, prompt)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(extractText(result))
	if text == "" {
		return errors.New("OpenAI response did not contain output text")
	}
	if err := os.MkdirAll(params.Work.Outputs, 0o755); err != nil {
		return err
	}
	usage := usageFor(result.Usage)
	writeNotification("fbt/event", map[string]any{
		"request_id":       req.ID,
		"transform_run_id": params.TransformRunID,
		"time":             time.Now().UTC().Format(time.RFC3339),
		"event_type":       "usage",
		"level":            "info",
		"message":          "OpenAI Responses API request completed",
		"attributes": map[string]any{
			"gen_ai.provider.name":       provider,
			"gen_ai.request.model":       model,
			"gen_ai.response.id":         result.ID,
			"gen_ai.operation.name":      "responses.create",
			"fbt.runner.external_api":    true,
			"fbt.runner.provider":        "openai",
			"fbt.runner.elapsed_seconds": time.Since(start).Seconds(),
			"gen_ai.usage.input_tokens":  usage["gen_ai.usage.input_tokens"],
			"gen_ai.usage.output_tokens": usage["gen_ai.usage.output_tokens"],
			"fbt.usage.total_tokens":     usage["fbt.usage.total_tokens"],
		},
	})

	var candidates []map[string]any
	for _, output := range outputs {
		candidate, err := writeOutput(params.Work.Outputs, output, text)
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
		"provenance":       provenanceFor(model, provider, result.ID, params.Model),
		"warnings":         []string{},
	})
	return nil
}

func buildPrompt(params runParams) (string, error) {
	var builder strings.Builder
	projectRoot := projectRootFromWork(params.Work.Outputs)
	builder.WriteString("You are an fbt external runner. Generate only the requested artifact content.\n")
	builder.WriteString("Follow the format, style guide, policy, and evidence requirements exactly.\n")
	builder.WriteString("Do not include secrets or raw private notes except as summarized evidence.\n\n")
	builder.WriteString("## Transform\n\n")
	builder.WriteString("- name: " + params.Transform.Name + "\n")
	builder.WriteString("- type: " + params.Transform.Type + "\n\n")
	builder.WriteString("## Declared Outputs\n\n")
	for _, output := range params.Outputs {
		builder.WriteString(fmt.Sprintf("- %s (%s) -> %s\n", output.Name, output.ArtifactType, output.DeclaredPath))
	}
	builder.WriteString("\n## Assets\n\n")
	used := 0
	for _, asset := range params.Assets {
		if used >= maxContextBytes {
			break
		}
		path := stringField(asset, "absolute_path")
		if path == "" {
			path = stringField(asset, "path")
		}
		path = resolvePath(projectRoot, path)
		content, err := readPath(path, maxContextBytes-used)
		if err != nil {
			return "", fmt.Errorf("read asset %s: %w", stringField(asset, "name"), err)
		}
		used += len(content)
		builder.WriteString("### " + stringField(asset, "name") + "\n\n")
		builder.WriteString(content)
		builder.WriteString("\n\n")
	}
	builder.WriteString("## Source Evidence\n\n")
	for _, input := range params.Inputs {
		if used >= maxContextBytes {
			break
		}
		builder.WriteString("### " + stringField(input, "name") + "\n\n")
		paths := stringSlice(input["resolved_paths"])
		if len(paths) == 0 {
			if currentVersion, ok := input["current_version"].(map[string]any); ok {
				if path := stringField(currentVersion, "absolute_path"); path != "" {
					paths = append(paths, path)
				}
			}
		}
		for _, path := range paths {
			if used >= maxContextBytes {
				break
			}
			path = resolvePath(projectRoot, path)
			content, err := readPath(path, maxContextBytes-used)
			if err != nil {
				return "", fmt.Errorf("read input %s: %w", path, err)
			}
			used += len(content)
			builder.WriteString("#### " + path + "\n\n")
			builder.WriteString(content)
			builder.WriteString("\n\n")
		}
	}
	return builder.String(), nil
}

func projectRootFromWork(workOutputs string) string {
	slash := filepath.ToSlash(filepath.Clean(workOutputs))
	if idx := strings.Index(slash, "/.fbt/work/"); idx >= 0 {
		return filepath.FromSlash(slash[:idx])
	}
	if strings.HasPrefix(slash, ".fbt/work/") {
		return "."
	}
	return ""
}

func resolvePath(root, path string) string {
	if path == "" || filepath.IsAbs(path) || root == "" {
		return path
	}
	return filepath.Join(root, filepath.FromSlash(path))
}

func readPath(path string, remaining int) (string, error) {
	if remaining <= 0 {
		return "\n[truncated]\n", nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return readDirectory(path, remaining)
	}
	return readFile(path, remaining)
}

func readDirectory(root string, remaining int) (string, error) {
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if isReadableTextPath(path) {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(paths)
	var builder strings.Builder
	for _, path := range paths {
		if builder.Len() >= remaining {
			builder.WriteString("\n[truncated]\n")
			break
		}
		content, err := readFile(path, remaining-builder.Len())
		if err != nil {
			return "", err
		}
		rel, _ := filepath.Rel(root, path)
		builder.WriteString("##### " + filepath.ToSlash(rel) + "\n\n")
		builder.WriteString(content)
		builder.WriteString("\n\n")
	}
	return builder.String(), nil
}

func readFile(path string, remaining int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	limited := io.LimitReader(file, int64(remaining))
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	text := string(data)
	if len(text) >= remaining {
		text += "\n[truncated]\n"
	}
	return text, nil
}

func isReadableTextPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".json", ".jsonl", ".yaml", ".yml", ".csv", ".log":
		return true
	default:
		return false
	}
}

func callResponses(apiKey, model, input string) (responsesResult, error) {
	endpoint := os.Getenv("OPENAI_RESPONSES_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	payload := map[string]any{
		"model": model,
		"input": input,
		"store": false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return responsesResult{}, err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return responsesResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return responsesResult{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return responsesResult{}, err
	}
	var result responsesResult
	if err := json.Unmarshal(data, &result); err != nil {
		return responsesResult{}, fmt.Errorf("decode OpenAI response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(data))
		if result.Error != nil && result.Error.Message != "" {
			message = result.Error.Message
		}
		return responsesResult{}, fmt.Errorf("OpenAI request failed with status %d: %s", resp.StatusCode, message)
	}
	if result.Error != nil {
		return responsesResult{}, fmt.Errorf("OpenAI response error: %s", result.Error.Message)
	}
	return result, nil
}

func extractText(result responsesResult) string {
	var builder strings.Builder
	for _, item := range result.Output {
		for _, block := range item.Content {
			if block.Type == "output_text" || block.Type == "text" {
				builder.WriteString(block.Text)
				if !strings.HasSuffix(block.Text, "\n") {
					builder.WriteString("\n")
				}
			}
		}
	}
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

func usageFor(raw map[string]any) map[string]any {
	inputTokens := numberField(raw, "input_tokens")
	outputTokens := numberField(raw, "output_tokens")
	totalTokens := numberField(raw, "total_tokens")
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}
	return map[string]any{
		"gen_ai.usage.input_tokens":  inputTokens,
		"gen_ai.usage.output_tokens": outputTokens,
		"fbt.usage.total_tokens":     totalTokens,
	}
}

func provenanceFor(model, provider, responseID string, raw map[string]any) map[string]any {
	return map[string]any{
		"runner":                "fbt-runner-openai",
		"runner_version":        "0.1.0",
		"model_provider":        provider,
		"model":                 model,
		"response_id":           responseID,
		"model_parameters_hash": hashJSON(raw),
		"materials":             []any{},
	}
}

func stringField(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func stringValue(values map[string]any, key, fallback string) string {
	if value := stringField(values, key); value != "" {
		return value
	}
	return fallback
}

func stringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && text != "" {
			out = append(out, text)
		}
	}
	return out
}

func numberField(values map[string]any, key string) int {
	switch value := values[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}

func hashJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "sha256:"
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
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
