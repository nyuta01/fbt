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
	t.Setenv("FBT_TEST_RUNNER_TOKEN", "present")
	discovery := NewDiscovery(root, config.ProjectConfig{
		Runners: []config.RunnerConfig{
			{Name: "command.local", Type: "command", Protocol: "stdio_jsonrpc", Command: command, Args: []string{"--mode", "test"}, CWD: "work", Env: []string{"FBT_TEST_RUNNER_TOKEN"}},
		},
	})
	if err := os.Mkdir(filepath.Join(root, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err := discovery.Resolve("command.local")
	if err != nil {
		t.Fatalf("resolve runner: %v", err)
	}
	if resolved.Source != SourceProjectConfig {
		t.Fatalf("unexpected source: %s", resolved.Source)
	}
	if resolved.CWD != filepath.Join(root, "work") || len(resolved.Args) != 2 {
		t.Fatalf("unexpected invocation settings: %+v", resolved)
	}
	if HasErrors(Diagnose(resolved)) {
		t.Fatalf("expected command diagnostics to pass: %+v", Diagnose(resolved))
	}
}

func TestDiagnoseMissingRunnerEnv(t *testing.T) {
	root := t.TempDir()
	command := writeExecutable(t, root, "bin/runner")
	resolved := Resolved{Name: "command.local", Command: command, CommandPath: command, CWD: root, Env: []string{"FBT_MISSING_TEST_TOKEN"}}
	diagnostics := Diagnose(resolved)
	if !HasErrors(diagnostics) || !hasDiagnosticCode(diagnostics, "RUNNER_ENV_MISSING") {
		t.Fatalf("expected missing env diagnostic, got %+v", diagnostics)
	}
}

func TestResolveProjectPlugin(t *testing.T) {
	root := t.TempDir()
	writeExecutable(t, root, "plugins/openai/fbt-openai-runner")
	writeFile(t, root, "plugins/openai/fbt_plugin.yml", `name: fbt-openai
version: 0.1.0
protocol: stdio_jsonrpc
command: ./fbt-openai-runner
args: ["--profile", "fbt"]
cwd: .
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
	if len(resolved.Args) != 2 || resolved.CWD != filepath.Join(root, "plugins", "openai") {
		t.Fatalf("unexpected plugin invocation settings: %+v", resolved)
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
