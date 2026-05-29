package openaiadapter

import (
	"bytes"
	"context"
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

	"github.com/nyuta01/fbt/sdk/go/outputs"
	"github.com/nyuta01/fbt/sdk/go/protocol"
	"github.com/nyuta01/fbt/sdk/go/stdiojsonrpc"
)

const (
	version         = "0.1.0"
	defaultEndpoint = "https://api.openai.com/v1/responses"
	maxContextBytes = 180_000
)

type runParams struct {
	Mode           string                    `json:"mode"`
	TransformRunID string                    `json:"transform_run_id"`
	Transform      transformInfo             `json:"transform"`
	Inputs         []map[string]any          `json:"inputs"`
	Outputs        []protocol.DeclaredOutput `json:"outputs"`
	Assets         []map[string]any          `json:"assets"`
	Model          map[string]any            `json:"model"`
	Policy         map[string]any            `json:"policy"`
	Work           workInfo                  `json:"work"`
}

type transformInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
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

func Handler() stdiojsonrpc.Handler {
	return stdiojsonrpc.Handler{
		Initialize: func(context.Context, protocol.Request, *stdiojsonrpc.Writer) (any, error) {
			return initializeResult(), nil
		},
		Validate: func(context.Context, protocol.Request, *stdiojsonrpc.Writer) (any, error) {
			return map[string]any{"status": "success", "warnings": []string{}}, nil
		},
		RunTransform: runOpenAI,
		Cancel: func(context.Context, protocol.Request, *stdiojsonrpc.Writer) error {
			os.Exit(0)
			return nil
		},
	}
}

func initializeResult() protocol.InitializeResult {
	return protocol.InitializeResult{
		Runner: protocol.RunnerInfo{
			Name:     "fbt-runner-openai",
			Version:  version,
			Language: "go",
		},
		Protocol: protocol.ProtocolInfo{
			Version: protocol.Version,
			Framing: protocol.FramingJSONL,
		},
		Capabilities: protocol.Capabilities{
			TransformTypes:   []string{"llm"},
			ArtifactTypes:    []string{"markdown", "markdown_directory", "text", "directory"},
			StreamEvents:     true,
			ToolCallLog:      false,
			UsageReporting:   true,
			CostEstimation:   false,
			OutputCandidates: true,
			SupportsDryRun:   true,
			SupportsCancel:   true,
		},
	}
}

func runOpenAI(ctx context.Context, req protocol.Request, writer *stdiojsonrpc.Writer) (any, error) {
	var params runParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, err
	}
	if params.Transform.Name == "" {
		params.Transform.Name = "openai_transform"
	}
	declaredOutputs := params.Outputs
	if len(declaredOutputs) == 0 {
		declaredOutputs = []protocol.DeclaredOutput{{Name: "output", ArtifactType: "markdown"}}
	}
	model := stringValue(params.Model, "name", "gpt-5")
	provider := stringValue(params.Model, "provider", "openai")
	if params.Mode == "plan" || params.Mode == "dry_run" {
		return protocol.RunTransformResult{
			Status:         "success",
			TransformRunID: params.TransformRunID,
			Provenance:     provenanceFor(model, provider, "", params.Model),
			Warnings:       []string{"dry run only; no OpenAI request sent"},
		}, nil
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is required")
	}
	if params.Work.Outputs == "" {
		return nil, errors.New("work.outputs is required")
	}
	prompt, err := buildPrompt(params)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	result, err := callResponses(ctx, apiKey, model, prompt)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(extractText(result))
	if text == "" {
		return nil, errors.New("OpenAI response did not contain output text")
	}
	if err := os.MkdirAll(params.Work.Outputs, 0o755); err != nil {
		return nil, err
	}
	usage := usageFor(result.Usage)
	if err := writer.Notification(protocol.MethodEvent, protocol.Event{
		RequestID:      req.ID,
		TransformRunID: params.TransformRunID,
		Time:           time.Now().UTC().Format(time.RFC3339),
		EventType:      "usage",
		Level:          "info",
		Message:        "OpenAI Responses API request completed",
		Attributes: map[string]any{
			"gen_ai.provider.name":       provider,
			"gen_ai.request.model":       model,
			"gen_ai.response.id":         result.ID,
			"gen_ai.operation.name":      "responses.create",
			"fbt.runner.external_api":    os.Getenv("FBT_OPENAI_ADAPTER_FAKE_RESPONSE") == "",
			"fbt.runner.provider":        "openai",
			"fbt.runner.elapsed_seconds": time.Since(start).Seconds(),
			"gen_ai.usage.input_tokens":  usage["gen_ai.usage.input_tokens"],
			"gen_ai.usage.output_tokens": usage["gen_ai.usage.output_tokens"],
			"fbt.usage.total_tokens":     usage["fbt.usage.total_tokens"],
		},
	}); err != nil {
		return nil, err
	}

	var candidates []map[string]any
	for _, output := range declaredOutputs {
		candidate, err := outputs.WriteText(params.Work.Outputs, output, []byte(text+"\n"))
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	outputItems := anySlice(candidates)
	if err := writer.Notification(protocol.MethodOutputCandidate, protocol.OutputCandidate{
		RequestID:      req.ID,
		TransformRunID: params.TransformRunID,
		Outputs:        outputItems,
	}); err != nil {
		return nil, err
	}
	return protocol.RunTransformResult{
		Status:         "success",
		TransformRunID: params.TransformRunID,
		Outputs:        outputItems,
		Usage:          usage,
		Provenance:     provenanceFor(model, provider, result.ID, params.Model),
		Warnings:       []string{},
	}, nil
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

func callResponses(ctx context.Context, apiKey, model, input string) (responsesResult, error) {
	if fake := os.Getenv("FBT_OPENAI_ADAPTER_FAKE_RESPONSE"); fake != "" {
		return responsesResult{
			ID: "resp_fbt_fake",
			Output: []responseItem{{
				Type: "message",
				Content: []responseBlock{{
					Type: "output_text",
					Text: fake,
				}},
			}},
			Usage: map[string]any{"input_tokens": 1, "output_tokens": 1, "total_tokens": 2},
		}, nil
	}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
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
		"runner_version":        version,
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

func anySlice(values []map[string]any) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}
