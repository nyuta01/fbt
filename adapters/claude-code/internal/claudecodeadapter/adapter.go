package claudecodeadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nyuta01/fbt/sdk/go/outputs"
	"github.com/nyuta01/fbt/sdk/go/protocol"
	"github.com/nyuta01/fbt/sdk/go/stdiojsonrpc"
)

const (
	version             = "0.1.0"
	stagedInputMaxBytes = 2 * 1024 * 1024
)

type runParams struct {
	Mode           string                    `json:"mode"`
	TransformRunID string                    `json:"transform_run_id"`
	Transform      map[string]any            `json:"transform"`
	Inputs         []map[string]any          `json:"inputs"`
	Outputs        []protocol.DeclaredOutput `json:"outputs"`
	Assets         []map[string]any          `json:"assets"`
	Model          map[string]any            `json:"model"`
	Policy         map[string]any            `json:"policy"`
	Work           struct {
		Root    string `json:"root"`
		Temp    string `json:"temp"`
		Outputs string `json:"outputs"`
	} `json:"work"`
}

type policyMapping struct {
	AllowedTools    []string
	DisallowedTools []string
	NoTools         bool
	MaxBudgetUSD    string
	PermissionMode  string
	Timeout         time.Duration
}

func Handler() stdiojsonrpc.Handler {
	return stdiojsonrpc.Handler{
		Initialize: func(context.Context, protocol.Request, *stdiojsonrpc.Writer) (any, error) {
			return initializeResult(), nil
		},
		Validate: func(context.Context, protocol.Request, *stdiojsonrpc.Writer) (any, error) {
			return map[string]any{"status": "success", "warnings": []string{}}, nil
		},
		RunTransform: runTransform,
		Cancel: func(context.Context, protocol.Request, *stdiojsonrpc.Writer) error {
			os.Exit(0)
			return nil
		},
	}
}

func initializeResult() protocol.InitializeResult {
	return protocol.InitializeResult{
		Runner: protocol.RunnerInfo{Name: "fbt-runner-claude-code", Version: version, Language: "go"},
		Protocol: protocol.ProtocolInfo{
			Version: protocol.Version,
			Framing: protocol.FramingJSONL,
		},
		Capabilities: protocol.Capabilities{
			TransformTypes:   []string{"agent"},
			ArtifactTypes:    []string{"markdown", "markdown_directory", "text", "directory"},
			StreamEvents:     true,
			ToolCallLog:      false,
			UsageReporting:   false,
			OutputCandidates: true,
			SupportsDryRun:   true,
			SupportsCancel:   true,
		},
	}
}

func runTransform(ctx context.Context, req protocol.Request, writer *stdiojsonrpc.Writer) (any, error) {
	var params runParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, err
	}
	declaredOutputs := params.Outputs
	if len(declaredOutputs) == 0 {
		declaredOutputs = []protocol.DeclaredOutput{{Name: "output", ArtifactType: "markdown"}}
	}
	if params.Mode == "plan" || params.Mode == "dry_run" {
		return protocol.RunTransformResult{
			Status:         "success",
			TransformRunID: params.TransformRunID,
			Warnings:       []string{"dry run only; Claude Code was not invoked"},
		}, nil
	}
	if params.Work.Root == "" || params.Work.Outputs == "" {
		return nil, errors.New("work.root and work.outputs are required")
	}
	mapping, err := mapPolicy(params.Policy)
	if err != nil {
		return nil, err
	}
	staging := filepath.Join(params.Work.Root, "staging", "claude-code")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(params.Work.Outputs, 0o755); err != nil {
		return nil, err
	}
	if err := materializeContext(staging, params); err != nil {
		return nil, err
	}
	if err := writer.Notification(protocol.MethodEvent, protocol.Event{
		RequestID:      req.ID,
		TransformRunID: params.TransformRunID,
		Time:           time.Now().UTC().Format(time.RFC3339),
		EventType:      "progress",
		Level:          "info",
		Message:        "running Claude Code adapter",
		Attributes: map[string]any{
			"fbt.adapter.staging_workspace":  staging,
			"fbt.adapter.policy_mode":        "fail_closed",
			"fbt.adapter.policy_fail_closed": true,
			"fbt.adapter.permission_mode":    mapping.PermissionMode,
			"fbt.adapter.allowed_tools":      mapping.AllowedTools,
			"fbt.adapter.disallowed_tools":   mapping.DisallowedTools,
			"fbt.runner.provider":            "claude-code",
		},
	}); err != nil {
		return nil, err
	}
	start := time.Now()
	runCtx := ctx
	if mapping.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, mapping.Timeout)
		defer cancel()
	}
	text, err := runClaude(runCtx, staging, buildPrompt(params, staging), runnableModel(params.Model), mapping)
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("Claude Code produced empty output")
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
		Provenance: map[string]any{
			"runner":                  "fbt-runner-claude-code",
			"runner_version":          version,
			"agent":                   "claude-code",
			"staging_workspace":       staging,
			"elapsed_seconds":         time.Since(start).Seconds(),
			"policy_mapping":          "fail_closed",
			"permission_mode":         mapping.PermissionMode,
			"allowed_tools":           mapping.AllowedTools,
			"disallowed_tools":        mapping.DisallowedTools,
			"model":                   stringValue(params.Model, "name", ""),
			"model_provider":          stringValue(params.Model, "provider", "anthropic"),
			"output_capture_method":   "stdout-text",
			"official_adapter_module": "adapters/claude-code",
		},
		Warnings: []string{},
	}, nil
}

func runClaude(ctx context.Context, staging, prompt, model string, mapping policyMapping) (string, error) {
	command := os.Getenv("FBT_CLAUDE_CODE_COMMAND")
	if command == "" {
		command = "claude"
	}
	args := []string{
		"-p",
		"--bare",
		"--output-format", "text",
		"--permission-mode", mapping.PermissionMode,
	}
	if mapping.NoTools {
		args = append(args, "--tools", "")
	}
	if len(mapping.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(mapping.AllowedTools, ","))
	}
	if len(mapping.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(mapping.DisallowedTools, ","))
	}
	if mapping.MaxBudgetUSD != "" {
		args = append(args, "--max-budget-usd", mapping.MaxBudgetUSD)
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, prompt)
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = staging
	cmd.Env = os.Environ()
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Claude Code failed: %w: %s", err, trimCommandOutput(stdout))
	}
	return string(stdout), nil
}

func mapPolicy(policy map[string]any) (policyMapping, error) {
	mapping := policyMapping{PermissionMode: "dontAsk"}
	if len(policy) == 0 {
		return mapping, nil
	}
	if network, ok := boolField(policy, "network"); ok && !network {
		return mapping, errors.New("policy denies network, but Claude Code adapter cannot enforce network isolation")
	}
	tools := mapField(policy, "tools")
	allow, allowSet := toolList(tools, "allow", "allowed")
	deny, denySet := toolList(tools, "deny", "denied")
	if allowSet {
		mapped, err := claudeTools(allow)
		if err != nil {
			return mapping, err
		}
		if len(mapped) == 0 {
			mapping.NoTools = true
		} else {
			mapping.AllowedTools = mapped
		}
	}
	if denySet {
		mapped, err := claudeTools(deny)
		if err != nil {
			return mapping, err
		}
		mapping.DisallowedTools = mapped
	}
	limits := mapField(policy, "limits")
	if seconds := numberValue(limits, "timeout_seconds"); seconds > 0 {
		mapping.Timeout = time.Duration(seconds) * time.Second
	}
	if calls := numberValue(limits, "max_tool_calls"); calls > 0 {
		return mapping, errors.New("Claude Code adapter cannot enforce max_tool_calls")
	}
	if cost, ok := numberText(limits, "max_cost_usd"); ok {
		mapping.MaxBudgetUSD = cost
	}
	return mapping, nil
}

func claudeTools(values []string) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		for _, tool := range claudeToolNames(value) {
			if tool == "" {
				return nil, fmt.Errorf("Claude Code adapter cannot map fbt tool policy %q", value)
			}
			if _, ok := seen[tool]; ok {
				continue
			}
			seen[tool] = struct{}{}
			out = append(out, tool)
		}
	}
	sort.Strings(out)
	return out, nil
}

func claudeToolNames(value string) []string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "shell", "bash":
		return []string{"Bash"}
	case "read_source", "read_artifact", "read_file":
		return []string{"Read"}
	case "search_project":
		return []string{"Glob", "Grep"}
	case "write_markdown", "write_artifact", "write_file":
		return []string{"Write"}
	case "write_source_files":
		return []string{"Edit", "Write"}
	default:
		return []string{""}
	}
}

func materializeContext(staging string, params runParams) error {
	if err := writeJSON(filepath.Join(staging, "fbt_request.json"), params); err != nil {
		return err
	}
	for _, input := range params.Inputs {
		if err := copyNamedPaths(filepath.Join(staging, "inputs"), stringField(input, "name"), stringSlice(input["resolved_paths"])); err != nil {
			return err
		}
	}
	for _, asset := range params.Assets {
		path := stringField(asset, "absolute_path")
		if path == "" {
			path = stringField(asset, "path")
		}
		if err := copyNamedPaths(filepath.Join(staging, "assets"), stringField(asset, "name"), []string{path}); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(staging, "README.md"), []byte("Use the staged fbt_request.json, inputs, and assets to generate the requested artifact. Return only the final artifact content.\n"), 0o644)
}

func copyNamedPaths(root, name string, paths []string) error {
	if name == "" {
		name = "item"
	}
	for i, path := range paths {
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		destRoot := filepath.Join(root, safeName(name), fmt.Sprintf("%03d", i+1))
		if info.IsDir() {
			if err := copyTextDirectory(path, destRoot); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(destRoot, 0o755); err != nil {
			return err
		}
		if err := copyFile(path, filepath.Join(destRoot, filepath.Base(path))); err != nil {
			return err
		}
	}
	return nil
}

func copyTextDirectory(src, dest string) error {
	var paths []string
	if err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if err := copyFile(path, filepath.Join(dest, rel)); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.Size() > stagedInputMaxBytes {
		return fmt.Errorf("staged input %q is %d bytes, exceeds adapter staging limit of %d bytes", src, info.Size(), stagedInputMaxBytes)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func buildPrompt(params runParams, staging string) string {
	name := stringField(params.Transform, "name")
	if name == "" {
		name = "fbt_agent_transform"
	}
	var builder strings.Builder
	builder.WriteString("Generate the fbt artifact for transform " + name + ".\n")
	builder.WriteString("Use only the staged files under this workspace.\n")
	builder.WriteString("Workspace: " + staging + "\n")
	builder.WriteString("Return only the final artifact content. Do not include tool logs, secrets, or explanations.\n")
	return builder.String()
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

func boolField(values map[string]any, key string) (bool, bool) {
	value, ok := values[key].(bool)
	return value, ok
}

func mapField(values map[string]any, key string) map[string]any {
	if value, ok := values[key].(map[string]any); ok {
		return value
	}
	return nil
}

func toolList(values map[string]any, keys ...string) ([]string, bool) {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		return stringSlice(value), true
	}
	return nil, false
}

func numberValue(values map[string]any, key string) int64 {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	case uint64:
		return int64(value)
	}
	return 0
}

func numberText(values map[string]any, key string) (string, bool) {
	if values == nil {
		return "", false
	}
	switch value := values[key].(type) {
	case int:
		return fmt.Sprintf("%d", value), true
	case int64:
		return fmt.Sprintf("%d", value), true
	case float64:
		return fmt.Sprintf("%g", value), true
	case uint64:
		return fmt.Sprintf("%d", value), true
	case string:
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func runnableModel(values map[string]any) string {
	if stringField(values, "provider") == "conformance" {
		return ""
	}
	model := stringField(values, "name")
	if model == "fixture" {
		return ""
	}
	return model
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

func safeName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "item"
	}
	value = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, value)
	return value
}

func anySlice(values []map[string]any) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func trimCommandOutput(data []byte) string {
	text := strings.TrimSpace(string(data))
	if len(text) > 4000 {
		return text[:4000] + "\n[truncated]"
	}
	return text
}
