package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nyuta01/fbt/internal/config"
)

func TestResolveProjectConfigCommand(t *testing.T) {
	root := t.TempDir()
	command := writeExecutable(t, root, "bin/runner")
	discovery := NewDiscovery(root, config.ProjectConfig{
		Runners: []config.RunnerConfig{
			{Name: "command.local", Type: "command", Protocol: "stdio_jsonrpc", Command: command},
		},
	})
	resolved, err := discovery.Resolve("command.local")
	if err != nil {
		t.Fatalf("resolve runner: %v", err)
	}
	if resolved.Source != SourceProjectConfig {
		t.Fatalf("unexpected source: %s", resolved.Source)
	}
	if HasErrors(Diagnose(resolved)) {
		t.Fatalf("expected command diagnostics to pass: %+v", Diagnose(resolved))
	}
}

func TestResolveProjectPlugin(t *testing.T) {
	root := t.TempDir()
	writeExecutable(t, root, "plugins/openai/fbt-openai-runner")
	writeFile(t, root, "plugins/openai/fbt_plugin.yml", `name: fbt-openai
version: 0.1.0
protocol: stdio_jsonrpc
command: ./fbt-openai-runner
provides:
  - runner: openai.responses
    type: llm
`)
	resolved, err := NewDiscovery(root, config.ProjectConfig{}).Resolve("openai.responses")
	if err != nil {
		t.Fatalf("resolve plugin: %v", err)
	}
	if resolved.Source != SourceProjectPlugin {
		t.Fatalf("unexpected source: %s", resolved.Source)
	}
	if resolved.PluginName != "fbt-openai" {
		t.Fatalf("unexpected plugin: %s", resolved.PluginName)
	}
}

func TestResolvePATHConvention(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	writeExecutable(t, bin, "fbt-runner-openai-responses")
	t.Setenv("PATH", bin)
	resolved, err := NewDiscovery(root, config.ProjectConfig{}).Resolve("openai.responses")
	if err != nil {
		t.Fatalf("resolve PATH runner: %v", err)
	}
	if resolved.Source != SourcePATH {
		t.Fatalf("unexpected source: %s", resolved.Source)
	}
}

func TestResolveMissingRunner(t *testing.T) {
	_, err := NewDiscovery(t.TempDir(), config.ProjectConfig{}).Resolve("missing.runner")
	if err == nil {
		t.Fatal("expected missing runner error")
	}
}

func writeExecutable(t *testing.T, root, relative string) string {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
