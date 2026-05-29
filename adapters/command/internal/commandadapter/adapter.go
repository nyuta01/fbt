package commandadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nyuta01/fbt/sdk/go/outputs"
	"github.com/nyuta01/fbt/sdk/go/protocol"
	"github.com/nyuta01/fbt/sdk/go/stdiojsonrpc"
)

const version = "0.1.0"

type runParams struct {
	Mode           string `json:"mode"`
	TransformRunID string `json:"transform_run_id"`
	Transform      struct {
		Command []string `json:"command"`
	} `json:"transform"`
	Work struct {
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
		Runner: protocol.RunnerInfo{
			Name:     "fbt-runner-command",
			Version:  version,
			Language: "go",
		},
		Protocol: protocol.ProtocolInfo{
			Version: protocol.Version,
			Framing: protocol.FramingJSONL,
		},
		Capabilities: protocol.Capabilities{
			TransformTypes:   []string{"command"},
			ArtifactTypes:    []string{"text", "markdown", "markdown_directory", "directory", "pdf"},
			StreamEvents:     true,
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
	command := params.Transform.Command
	if len(command) == 0 {
		command = defaultCommand()
	}
	if len(command) == 0 {
		return nil, fmt.Errorf("transform.command is required")
	}
	if params.Work.Outputs == "" {
		return nil, fmt.Errorf("work.outputs is required")
	}
	if params.Mode == "plan" || params.Mode == "dry_run" {
		return protocol.RunTransformResult{
			Status:         "success",
			TransformRunID: params.TransformRunID,
			Warnings:       []string{"dry run only; command was not executed"},
		}, nil
	}
	if err := os.MkdirAll(params.Work.Outputs, 0o755); err != nil {
		return nil, err
	}
	start := time.Now()
	if err := writer.Notification(protocol.MethodEvent, protocol.Event{
		RequestID:      req.ID,
		TransformRunID: params.TransformRunID,
		Time:           time.Now().UTC().Format(time.RFC3339),
		EventType:      "progress",
		Level:          "info",
		Message:        "running command adapter",
		Attributes: map[string]any{
			"fbt.runner.command": filepath.Base(command[0]),
		},
	}); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	if dir := os.Getenv("FBT_COMMAND_WORKDIR"); dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(),
		"FBT_WORK_ROOT="+params.Work.Root,
		"FBT_WORK_TEMP="+params.Work.Temp,
		"FBT_WORK_OUTPUTS="+params.Work.Outputs,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	candidates, err := outputs.Collect(params.Work.Outputs)
	if err != nil {
		return nil, err
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
		Usage: map[string]any{
			"fbt.runner.elapsed_seconds": time.Since(start).Seconds(),
		},
		Warnings: []string{},
	}, nil
}

func defaultCommand() []string {
	if command := os.Getenv("FBT_COMMAND_ADAPTER_DEFAULT_COMMAND"); command != "" {
		return []string{command}
	}
	return nil
}

func anySlice(values []map[string]any) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}
