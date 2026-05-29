package codexcliadapter

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

const version = "0.1.0"

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
		Runner: protocol.RunnerInfo{Name: "fbt-runner-codex-cli", Version: version, Language: "go"},
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
			Warnings:       []string{"dry run only; Codex CLI was not invoked"},
		}, nil
	}
	if params.Work.Root == "" || params.Work.Outputs == "" {
		return nil, errors.New("work.root and work.outputs are required")
	}
	staging := filepath.Join(params.Work.Root, "staging", "codex-cli")
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
		Message:        "running Codex CLI adapter",
		Attributes: map[string]any{
			"fbt.adapter.staging_workspace":  staging,
			"fbt.adapter.policy_mode":        "fail_closed",
			"fbt.adapter.policy_fail_closed": true,
			"fbt.runner.provider":            "codex-cli",
		},
	}); err != nil {
		return nil, err
	}
	start := time.Now()
	text, err := runCodex(ctx, staging, buildPrompt(params, staging), runnableModel(params.Model))
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("Codex CLI produced empty output")
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
			"runner":                  "fbt-runner-codex-cli",
			"runner_version":          version,
			"agent":                   "codex-cli",
			"staging_workspace":       staging,
			"elapsed_seconds":         time.Since(start).Seconds(),
			"policy_mapping":          "fail_closed",
			"model":                   stringValue(params.Model, "name", ""),
			"model_provider":          stringValue(params.Model, "provider", "openai"),
			"output_capture_method":   "output-last-message-or-stdout",
			"official_adapter_module": "adapters/codex-cli",
		},
		Warnings: []string{},
	}, nil
}

func runCodex(ctx context.Context, staging, prompt, model string) (string, error) {
	command := os.Getenv("FBT_CODEX_CLI_COMMAND")
	if command == "" {
		command = "codex"
	}
	lastMessage := filepath.Join(staging, "codex-last-message.txt")
	args := []string{
		"exec",
		"--cd", staging,
		"--sandbox", "workspace-write",
		"--skip-git-repo-check",
		"--ephemeral",
		"--ignore-user-config",
		"--ignore-rules",
		"--output-last-message", lastMessage,
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, prompt)
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = staging
	cmd.Env = os.Environ()
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Codex CLI failed: %w: %s", err, trimCommandOutput(combined))
	}
	if data, err := os.ReadFile(lastMessage); err == nil && len(data) > 0 {
		return string(data), nil
	}
	return string(combined), nil
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
	_, err = io.Copy(out, io.LimitReader(in, 2*1024*1024))
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
