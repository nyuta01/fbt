package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"
)

type Options struct {
	ProjectName string
	Destination string
	Template    string
	Force       bool
	RunnerRoot  string
}

type Result struct {
	ProjectName string   `json:"project_name"`
	ProjectDir  string   `json:"project_dir"`
	Template    string   `json:"template"`
	Files       []string `json:"files"`
}

type fileSpec struct {
	Path       string
	Content    string
	Executable bool
}

func CreateProject(options Options) (Result, error) {
	if options.Template == "" {
		options.Template = "blank"
	}
	if options.Template == "knowledge_ops" {
		options.Template = "support"
	}
	if options.ProjectName == "" {
		options.ProjectName = "fbt_project"
	}
	projectName := normalizeName(filepath.Base(options.ProjectName))
	destination := options.Destination
	if destination == "" {
		destination = options.ProjectName
	}
	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return Result{}, err
	}
	runnerRoot := options.RunnerRoot
	if runnerRoot == "" {
		runnerRoot = defaultRunnerRoot()
	}

	files, err := templateFiles(options.Template, projectName, runnerRoot)
	if err != nil {
		return Result{}, err
	}
	result := Result{ProjectName: projectName, ProjectDir: absDestination, Template: options.Template}
	for _, file := range files {
		target := filepath.Join(absDestination, filepath.FromSlash(file.Path))
		if !options.Force {
			if _, err := os.Stat(target); err == nil {
				return Result{}, fmt.Errorf("refusing to overwrite existing file: %s", target)
			} else if !os.IsNotExist(err) {
				return Result{}, err
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return Result{}, err
		}
		mode := os.FileMode(0o644)
		if file.Executable {
			mode = 0o755
		}
		if err := os.WriteFile(target, []byte(file.Content), mode); err != nil {
			return Result{}, err
		}
		result.Files = append(result.Files, filepath.ToSlash(file.Path))
	}
	sort.Strings(result.Files)
	return result, nil
}

func templateFiles(template, projectName, runnerRoot string) ([]fileSpec, error) {
	switch template {
	case "blank":
		return blankFiles(projectName), nil
	case "support":
		return supportFiles(projectName, runnerRoot), nil
	case "incident":
		return incidentFiles(projectName, runnerRoot), nil
	default:
		return nil, fmt.Errorf("unknown template %q", template)
	}
}

func blankFiles(projectName string) []fileSpec {
	return []fileSpec{
		{Path: "fs_project.yml", Content: fsProject(projectName, false, "")},
		{Path: "sources/.gitkeep", Content: ""},
		{Path: "transforms/.gitkeep", Content: ""},
		{Path: "assets/.gitkeep", Content: ""},
		{Path: "policies/.gitkeep", Content: ""},
		{Path: "evals/.gitkeep", Content: ""},
		{Path: "target/.gitkeep", Content: ""},
	}
}

func supportFiles(projectName, runnerRoot string) []fileSpec {
	files := runnableBaseFiles(projectName, runnerRoot, `selectors:
  - name: support_daily
    definition:
      method: tag
      value: support
`)
	files = append(files,
		fileSpec{Path: "data/support/tickets/2026-05-28.jsonl", Content: "{\"id\":\"T-1\",\"summary\":\"Login issue resolved\",\"impact\":\"One customer blocked\"}\n"},
		fileSpec{Path: "assets/support_style_guide.md", Content: "# Support Style Guide\n\n- Separate facts from assumptions.\n- Include next actions.\n"},
		fileSpec{Path: "assets/support.yml", Content: `assets:
  - name: support_style_guide
    type: style_guide
    path: assets/support_style_guide.md
`},
		fileSpec{Path: "sources/support.yml", Content: `sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
        tags: ["support", "raw"]
`},
		fileSpec{Path: "policies/support.yml", Content: `policies:
  - name: support_agent_scope
    read: ["data/support/", "target/artifacts/support/"]
    write: [".fbt/work/", "target/artifacts/support/"]
    network: false
    review:
      required: true
      group: support_leads
`},
		fileSpec{Path: "evals/support.yml", Content: `evals:
  - name: required_case_sections
    type: deterministic
    config:
      sections: ["Case Summaries"]
    grants_confidence: structural

  - name: required_agent_sections
    type: deterministic
    config:
      sections: ["Result"]
    grants_confidence: structural
`},
		fileSpec{Path: "transforms/support/case_summaries.yml", Content: `transforms:
  - name: case_summaries
    type: llm
    runner: demo.llm
    model:
      provider: demo
      name: deterministic-demo-llm
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    assets:
      - ref: support_style_guide
    policy: support_agent_scope
    evals:
      - required_case_sections
    review:
      required: true
      group: support_leads
    tags: ["support", "knowledge"]
`},
		fileSpec{Path: "transforms/support/weekly_insights.yml", Content: `transforms:
  - name: weekly_support_insights
    type: agent
    runner: demo.agent
    model:
      provider: demo
      name: deterministic-demo-agent
    inputs:
      - ref: case_summaries
        require:
          confidence: reviewed
          review:
            status: approved
    outputs:
      - name: weekly_support_insights
        type: markdown
        path: target/artifacts/support/weekly_insights.md
    assets:
      - ref: support_style_guide
    tools: ["read_artifact", "write_artifact"]
    policy: support_agent_scope
    evals:
      - required_agent_sections
    tags: ["support", "weekly"]
`},
		fileSpec{Path: "README.md", Content: "# " + projectName + "\n\nThis template uses deterministic demo runners (`demo.llm` and `demo.agent`) so the local control-plane loop works without provider credentials.\n\nRun `fbt plan`, `fbt build --select case_summaries`, `fbt review approve case_summaries`, then `fbt build --select weekly_support_insights`.\n\nReplace the demo runner entries in `fs_project.yml` with external runner commands before using real provider or agent execution.\n"},
	)
	return files
}

func incidentFiles(projectName, runnerRoot string) []fileSpec {
	files := runnableBaseFiles(projectName, runnerRoot, "")
	files = append(files,
		fileSpec{Path: "data/incident/logs/2026-05-28.log", Content: "10:00 service degraded\n10:05 rollback started\n10:15 service recovered\n"},
		fileSpec{Path: "assets/incident_style_guide.md", Content: "# Incident Style Guide\n\n- Capture timeline.\n- Separate impact, cause, and follow-up.\n"},
		fileSpec{Path: "assets/incident.yml", Content: `assets:
  - name: incident_style_guide
    type: style_guide
    path: assets/incident_style_guide.md
`},
		fileSpec{Path: "sources/incident.yml", Content: `sources:
  - name: incident
    artifacts:
      - name: raw_logs
        type: text
        path: data/incident/logs/2026-05-28.log
        tags: ["incident", "raw"]
`},
		fileSpec{Path: "policies/incident.yml", Content: `policies:
  - name: incident_scope
    read: ["data/incident/"]
    write: [".fbt/work/", "target/artifacts/incident/"]
    network: false
`},
		fileSpec{Path: "evals/incident.yml", Content: `evals:
  - name: required_timeline_sections
    type: deterministic
    config:
      sections: ["Incident Timeline"]
    grants_confidence: structural
`},
		fileSpec{Path: "transforms/incident/timeline.yml", Content: `transforms:
  - name: incident_timeline
    type: llm
    runner: demo.llm
    model:
      provider: demo
      name: deterministic-demo-llm
    inputs:
      - source: incident.raw_logs
    outputs:
      - name: incident_timeline
        type: markdown
        path: target/artifacts/incident/timeline.md
    assets:
      - ref: incident_style_guide
    policy: incident_scope
    evals:
      - required_timeline_sections
    tags: ["incident"]
`},
		fileSpec{Path: "README.md", Content: "# " + projectName + "\n\nThis template uses the deterministic `demo.llm` runner so the local control-plane loop works without provider credentials.\n\nRun `fbt build --select incident_timeline` to generate the demo incident timeline.\n\nReplace the demo runner entry in `fs_project.yml` with an external runner command before using real provider execution.\n"},
	)
	return files
}

func runnableBaseFiles(projectName, runnerRoot, selectors string) []fileSpec {
	return []fileSpec{
		{Path: "fs_project.yml", Content: fsProject(projectName, true, selectors)},
		{Path: "bin/fbt-demo-llm-runner", Content: runnerScript(filepath.Join(runnerRoot, "runners", "llm")), Executable: true},
		{Path: "bin/fbt-demo-agent-runner", Content: runnerScript(filepath.Join(runnerRoot, "runners", "agent")), Executable: true},
	}
}

func fsProject(projectName string, withRunners bool, selectors string) string {
	runners := ""
	if withRunners {
		runners = `runners:
  - name: demo.llm
    type: llm
    protocol: stdio_jsonrpc
    command: bin/fbt-demo-llm-runner
  - name: demo.agent
    type: agent
    protocol: stdio_jsonrpc
    command: bin/fbt-demo-agent-runner
` + selectors
	}
	return fmt.Sprintf(`name: %s
config_version: 1
version: 0.1.0
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]
artifact_path: "target/artifacts"
state:
  backend: local
  path: .fbt/state
%s`, projectName, runners)
}

func runnerScript(path string) string {
	runnerRoot := filepath.Dir(filepath.Dir(path))
	rel, err := filepath.Rel(runnerRoot, path)
	if err != nil {
		rel = path
	} else {
		rel = "./" + filepath.ToSlash(rel)
	}
	return fmt.Sprintf("#!/usr/bin/env sh\n# Deterministic demo runner wrapper generated by fbt init.\n# Replace the matching runner entry in fs_project.yml for real provider execution.\ncd %s || exit 1\nexec go run %s \"$@\"\n", shellQuote(runnerRoot), shellQuote(rel))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func defaultRunnerRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func normalizeName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "fbt_project"
	}
	var builder strings.Builder
	for i, r := range value {
		valid := r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
		if !valid {
			r = '_'
		}
		if i == 0 && !unicode.IsLetter(r) {
			builder.WriteByte('p')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
