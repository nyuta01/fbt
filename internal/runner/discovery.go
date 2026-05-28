package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nyuta01/fbt/internal/config"
	"github.com/nyuta01/fbt/internal/plugin"
)

type Source string

const (
	SourceProjectConfig Source = "project_config"
	SourceProjectPlugin Source = "project_plugin"
	SourceUserPlugin    Source = "user_plugin"
	SourcePATH          Source = "path"
)

type Diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

type Resolved struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Protocol    string         `json:"protocol"`
	Command     string         `json:"command"`
	Args        []string       `json:"args,omitempty"`
	CWD         string         `json:"cwd,omitempty"`
	CommandPath string         `json:"command_path,omitempty"`
	Source      Source         `json:"source"`
	PluginName  string         `json:"plugin_name,omitempty"`
	Version     string         `json:"version,omitempty"`
	Env         []string       `json:"env,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
}

type Discovery struct {
	ProjectDir string
	Config     config.ProjectConfig
	FBTHome    string
}

func NewDiscovery(projectDir string, cfg config.ProjectConfig) Discovery {
	return Discovery{ProjectDir: projectDir, Config: cfg}
}

func (d Discovery) Resolve(name string) (Resolved, error) {
	if resolved, ok, err := d.resolveProjectConfig(name); ok || err != nil {
		return resolved, err
	}
	if resolved, ok, err := d.resolvePlugin(name, filepath.Join(d.ProjectDir, "plugins"), SourceProjectPlugin); ok || err != nil {
		return resolved, err
	}
	if resolved, ok, err := d.resolvePlugin(name, filepath.Join(d.fbtHome(), "plugins"), SourceUserPlugin); ok || err != nil {
		return resolved, err
	}
	if commandPath, err := exec.LookPath(conventionalCommand(name)); err == nil {
		return Resolved{Name: name, Protocol: "stdio_jsonrpc", Command: conventionalCommand(name), CommandPath: commandPath, Source: SourcePATH}, nil
	}
	return Resolved{Name: name}, fmt.Errorf("runner not installed: %s", name)
}

func (d Discovery) List() ([]Resolved, error) {
	seen := map[string]Resolved{}
	for _, runner := range d.Config.Runners {
		resolved, _, err := d.resolveProjectConfig(runner.Name)
		if err != nil {
			return nil, err
		}
		seen[runner.Name] = resolved
	}
	for _, root := range []struct {
		path   string
		source Source
	}{
		{filepath.Join(d.ProjectDir, "plugins"), SourceProjectPlugin},
		{filepath.Join(d.fbtHome(), "plugins"), SourceUserPlugin},
	} {
		manifests, err := plugin.LoadAll(root.path)
		if err != nil {
			return nil, err
		}
		for _, manifest := range manifests {
			for _, provided := range manifest.Provides {
				if _, ok := seen[provided.Runner]; ok {
					continue
				}
				seen[provided.Runner] = resolvedFromPlugin(provided.Runner, manifest, provided, root.source)
			}
		}
	}
	out := make([]Resolved, 0, len(seen))
	for _, resolved := range seen {
		resolved.Diagnostics = Diagnose(resolved)
		out = append(out, resolved)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func Diagnose(resolved Resolved) []Diagnostic {
	var diagnostics []Diagnostic
	if resolved.Command == "" {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "RUNNER_COMMAND_MISSING", Message: "runner command is not configured"})
		return diagnostics
	}
	if resolved.CWD != "" {
		info, err := os.Stat(resolved.CWD)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "RUNNER_CWD_NOT_ACCESSIBLE", Message: err.Error()})
		} else if !info.IsDir() {
			diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "RUNNER_CWD_NOT_DIRECTORY", Message: fmt.Sprintf("runner cwd is not a directory: %s", resolved.CWD)})
		} else {
			diagnostics = append(diagnostics, Diagnostic{Severity: "info", Code: "RUNNER_CWD_OK", Message: fmt.Sprintf("runner cwd is accessible: %s", resolved.CWD)})
		}
	}
	for _, name := range resolved.Env {
		if _, ok := os.LookupEnv(name); !ok {
			diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "RUNNER_ENV_MISSING", Message: fmt.Sprintf("runner env %s is not set", name)})
		}
	}
	if len(resolved.Env) > 0 && !hasDiagnosticCode(diagnostics, "RUNNER_ENV_MISSING") {
		diagnostics = append(diagnostics, Diagnostic{Severity: "info", Code: "RUNNER_ENV_OK", Message: "declared runner environment is available"})
	}
	commandPath := resolved.CommandPath
	if commandPath == "" {
		if path, err := exec.LookPath(resolved.Command); err == nil {
			commandPath = path
		}
	}
	if commandPath == "" {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "RUNNER_COMMAND_NOT_FOUND", Message: fmt.Sprintf("runner command not found: %s", resolved.Command)})
		return diagnostics
	}
	info, err := os.Stat(commandPath)
	if err != nil {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "RUNNER_COMMAND_NOT_ACCESSIBLE", Message: err.Error()})
		return diagnostics
	}
	if info.Mode()&0o111 == 0 {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "RUNNER_COMMAND_NOT_EXECUTABLE", Message: fmt.Sprintf("runner command is not executable: %s", commandPath)})
		return diagnostics
	}
	diagnostics = append(diagnostics, Diagnostic{Severity: "info", Code: "RUNNER_COMMAND_OK", Message: fmt.Sprintf("runner command is executable: %s", commandPath)})
	return diagnostics
}

func HasErrors(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			return true
		}
	}
	return false
}

func (d Discovery) resolveProjectConfig(name string) (Resolved, bool, error) {
	var matches []config.RunnerConfig
	for _, runner := range d.Config.Runners {
		if runner.Name == name {
			matches = append(matches, runner)
		}
	}
	if len(matches) == 0 {
		return Resolved{}, false, nil
	}
	if len(matches) > 1 {
		return Resolved{}, true, fmt.Errorf("multiple project runner entries match %s", name)
	}
	runner := matches[0]
	commandPath := resolveConfiguredCommand(d.ProjectDir, runner.Command)
	return Resolved{
		Name:        runner.Name,
		Type:        runner.Type,
		Protocol:    runner.Protocol,
		Command:     runner.Command,
		Args:        append([]string(nil), runner.Args...),
		CWD:         resolveConfiguredCWD(d.ProjectDir, runner.CWD, ""),
		CommandPath: commandPath,
		Source:      SourceProjectConfig,
		Env:         append([]string(nil), runner.Env...),
		Config:      runner.Config,
	}, true, nil
}

func (d Discovery) resolvePlugin(name, root string, source Source) (Resolved, bool, error) {
	manifests, err := plugin.LoadAll(root)
	if err != nil {
		return Resolved{}, false, err
	}
	var matches []Resolved
	for _, manifest := range manifests {
		for _, provided := range manifest.Provides {
			if provided.Runner == name {
				matches = append(matches, resolvedFromPlugin(name, manifest, provided, source))
			}
		}
	}
	if len(matches) == 0 {
		return Resolved{}, false, nil
	}
	if len(matches) > 1 {
		return Resolved{}, true, fmt.Errorf("multiple plugin manifests match %s at %s", name, root)
	}
	return matches[0], true, nil
}

func resolvedFromPlugin(name string, manifest plugin.Manifest, provided plugin.ProvidedRunner, source Source) Resolved {
	commandPath := resolveConfiguredCommand(manifest.RootDir, manifest.Command)
	return Resolved{
		Name:        name,
		Type:        provided.Type,
		Protocol:    manifest.Protocol,
		Command:     manifest.Command,
		Args:        append([]string(nil), manifest.Args...),
		CWD:         resolveConfiguredCWD(manifest.RootDir, manifest.CWD, ""),
		CommandPath: commandPath,
		Source:      source,
		PluginName:  manifest.Name,
		Version:     manifest.Version,
		Env:         append([]string(nil), manifest.Env...),
	}
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func resolveConfiguredCommand(baseDir, command string) string {
	if command == "" {
		return ""
	}
	if filepath.IsAbs(command) {
		return command
	}
	if strings.ContainsRune(command, os.PathSeparator) {
		return filepath.Join(baseDir, command)
	}
	if path, err := exec.LookPath(command); err == nil {
		return path
	}
	return ""
}

func resolveConfiguredCWD(baseDir, cwd, fallback string) string {
	if cwd == "" {
		return fallback
	}
	if filepath.IsAbs(cwd) {
		return cwd
	}
	return filepath.Join(baseDir, cwd)
}

func (d Discovery) fbtHome() string {
	if d.FBTHome != "" {
		return d.FBTHome
	}
	if env := os.Getenv("FBT_HOME"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(d.ProjectDir, ".fbt")
	}
	return filepath.Join(home, ".fbt")
}

func conventionalCommand(name string) string {
	normalized := strings.NewReplacer(".", "-", "_", "-").Replace(name)
	return "fbt-runner-" + normalized
}
