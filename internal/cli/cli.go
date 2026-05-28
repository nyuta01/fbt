package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nyuta01/fbt/internal/artifact"
	buildmgr "github.com/nyuta01/fbt/internal/build"
	"github.com/nyuta01/fbt/internal/config"
	diffmgr "github.com/nyuta01/fbt/internal/diff"
	"github.com/nyuta01/fbt/internal/graph"
	"github.com/nyuta01/fbt/internal/lineage"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/parser"
	"github.com/nyuta01/fbt/internal/planner"
	"github.com/nyuta01/fbt/internal/protocol"
	runnermgr "github.com/nyuta01/fbt/internal/runner"
	"github.com/nyuta01/fbt/internal/state"
	"github.com/nyuta01/fbt/internal/telemetry"
	"github.com/nyuta01/fbt/internal/templates"
	versioninfo "github.com/nyuta01/fbt/internal/version"
)

type commandHelp struct {
	Name        string
	Description string
}

var primaryCommands = []commandHelp{
	{"init", "Create a project"},
	{"doctor", "Check project and runner readiness"},
	{"plan", "Preview run, skip, and blocked transforms"},
	{"build", "Build selected artifacts and write receipts"},
	{"artifact", "Inspect artifact paths, versions, and lineage"},
	{"diff", "Compare artifact versions"},
	{"export", "Write standard lineage or trace records"},
}

type options struct {
	ProjectDir string
	StateDir   string
	JSON       bool
	Select     string
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return 0
	}

	opts, commandArgs, err := parseOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	if len(commandArgs) == 0 {
		printHelp(stdout)
		return 0
	}

	switch commandArgs[0] {
	case "version", "--version", "-v":
		if err := rejectSelect("version", opts); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		if err := expectNoArgs("version", commandArgs[1:]); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		info := versioninfo.Current()
		if opts.JSON {
			writeJSON(stdout, map[string]any{
				"command":    "version",
				"status":     "success",
				"version":    info.Version,
				"commit":     info.Commit,
				"build_date": info.BuildDate,
			})
			return 0
		}
		fmt.Fprintf(stdout, "fbt %s\n", info.Version)
		return 0
	case "init":
		return runInit(opts, commandArgs[1:], stdout, stderr)
	case "plan":
		return runPlan(opts, commandArgs[1:], stdout, stderr)
	case "build":
		return runBuild(opts, commandArgs[1:], stdout, stderr)
	case "diff":
		return runDiff(opts, commandArgs[1:], stdout, stderr)
	case "artifact":
		return runArtifact(opts, commandArgs[1:], stdout, stderr)
	case "doctor":
		return runDoctor(opts, commandArgs[1:], stdout, stderr)
	case "export":
		return runExport(opts, commandArgs[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", commandArgs[0])
		printHelp(stderr)
		return 2
	}
}

func expectNoArgs(command string, args []string) error {
	if len(args) == 0 {
		return nil
	}
	arg := args[0]
	if strings.HasPrefix(arg, "-") {
		return fmt.Errorf("unknown %s flag: %s", command, arg)
	}
	return fmt.Errorf("%s accepts no arguments", command)
}

func rejectSelect(command string, opts options) error {
	if opts.Select == "" {
		return nil
	}
	return fmt.Errorf("%s does not accept --select", command)
}

func expectArgs(command string, args []string, count int) error {
	if len(args) < count {
		return fmt.Errorf("%s requires %d argument(s)", command, count)
	}
	if len(args) > count {
		arg := args[count]
		if strings.HasPrefix(arg, "-") {
			return fmt.Errorf("unknown %s flag: %s", command, arg)
		}
		return fmt.Errorf("%s accepts %d argument(s)", command, count)
	}
	return nil
}

func expectAtMostArgs(command string, args []string, count int) error {
	if len(args) <= count {
		return nil
	}
	arg := args[count]
	if strings.HasPrefix(arg, "-") {
		return fmt.Errorf("unknown %s flag: %s", command, arg)
	}
	return fmt.Errorf("%s accepts at most %d argument(s)", command, count)
}

func runDiff(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := rejectSelect("diff", opts); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	diffOpts, err := parseDiffArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	snapshot, err := ctx.Store.ReadState()
	if err != nil {
		printError("diff", err, stderr, opts.JSON)
		return 5
	}
	versions, err := ctx.Store.ReadArtifactVersions()
	if err != nil {
		printError("diff", err, stderr, opts.JSON)
		return 5
	}
	current, ok := findVersion(snapshot, versions, diffOpts.Target)
	if !ok {
		fmt.Fprintf(stderr, "Error: artifact not found: %s\n", diffOpts.Target)
		return 2
	}
	against, err := resolveAgainst(ctx.Store, snapshot, versions, current, diffOpts.Against)
	if err != nil {
		printError("diff", err, stderr, opts.JSON)
		return 2
	}
	result, err := diffmgr.ComparePaths(filepath.Join(ctx.ParseResult.ProjectDir, against.StoragePath), filepath.Join(ctx.ParseResult.ProjectDir, current.StoragePath), against.VersionID, current.VersionID)
	if err != nil {
		printError("diff", err, stderr, opts.JSON)
		return 1
	}
	if opts.JSON {
		writeJSON(stdout, map[string]any{"command": "diff", "status": "success", "diff": result})
		return 0
	}
	for _, section := range result.Sections {
		fmt.Fprintf(stdout, "%s: %s\n", section.Status, section.Heading)
	}
	fmt.Fprint(stdout, result.Unified)
	return 0
}

type diffOptions struct {
	Target  string
	Against string
}

func parseDiffArgs(args []string) (diffOptions, error) {
	opts := diffOptions{Against: "previous"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--against":
			i++
			if i >= len(args) {
				return diffOptions{}, fmt.Errorf("--against requires a value")
			}
			opts.Against = args[i]
		case strings.HasPrefix(arg, "--against="):
			opts.Against = strings.TrimPrefix(arg, "--against=")
		case strings.HasPrefix(arg, "--"):
			return diffOptions{}, fmt.Errorf("unknown diff flag: %s", arg)
		default:
			if opts.Target != "" {
				return diffOptions{}, fmt.Errorf("diff accepts one target")
			}
			opts.Target = arg
		}
	}
	if opts.Target == "" {
		return diffOptions{}, fmt.Errorf("diff requires a target")
	}
	return opts, nil
}

func runExport(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := rejectSelect("export", opts); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	exportOpts, err := parseExportArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	switch exportOpts.Format {
	case "openlineage":
		return runExportOpenLineage(opts, exportOpts, stdout, stderr)
	case "otel":
		return runExportOTel(opts, exportOpts, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown export format: %s\n", exportOpts.Format)
		return 2
	}
}

func runExportOpenLineage(opts options, exportOpts exportOptions, stdout io.Writer, stderr io.Writer) int {
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	snapshot, err := ctx.Store.ReadState()
	if err != nil {
		printError("export openlineage", err, stderr, opts.JSON)
		return 5
	}
	versions, err := ctx.Store.ReadArtifactVersions()
	if err != nil {
		printError("export openlineage", err, stderr, opts.JSON)
		return 5
	}
	evaluations, err := ctx.Store.ReadEvaluationResults()
	if err != nil {
		printError("export openlineage", err, stderr, opts.JSON)
		return 5
	}
	events := lineage.OpenLineageEvents(lineage.OpenLineageInput{
		Manifest:          ctx.Manifest,
		Snapshot:          snapshot,
		ArtifactVersions:  versions,
		EvaluationResults: evaluations,
	})
	if exportOpts.OutputPath != "" {
		if err := writeOpenLineageOutput(exportOpts.OutputPath, events); err != nil {
			printError("export openlineage", err, stderr, opts.JSON)
			return 5
		}
		if opts.JSON {
			writeJSON(stdout, map[string]any{
				"command":     "export openlineage",
				"status":      "success",
				"format":      "openlineage",
				"events":      len(events),
				"output_path": exportOpts.OutputPath,
			})
			return 0
		}
		fmt.Fprintf(stdout, "OpenLineage events written to %s\n", exportOpts.OutputPath)
		fmt.Fprintf(stdout, "Events: %d\n", len(events))
		return 0
	}
	if opts.JSON {
		writeJSON(stdout, map[string]any{
			"command": "export openlineage",
			"status":  "success",
			"format":  "openlineage",
			"events":  len(events),
			"records": events,
		})
		return 0
	}
	if err := lineage.WriteOpenLineageNDJSON(stdout, events); err != nil {
		printError("export openlineage", err, stderr, opts.JSON)
		return 5
	}
	return 0
}

func runExportOTel(opts options, exportOpts exportOptions, stdout io.Writer, stderr io.Writer) int {
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	runResults, err := ctx.Store.ReadRunResults()
	if err != nil {
		printError("export otel", err, stderr, opts.JSON)
		return 5
	}
	versions, err := ctx.Store.ReadArtifactVersions()
	if err != nil {
		printError("export otel", err, stderr, opts.JSON)
		return 5
	}
	payload := telemetry.OTLPTraces(telemetry.OTLPInput{
		Manifest:         ctx.Manifest,
		RunResults:       runResults,
		ArtifactVersions: versions,
		FBTVersion:       versioninfo.Version,
	})
	if exportOpts.OutputPath != "" {
		if err := writeOTelOutput(exportOpts.OutputPath, payload); err != nil {
			printError("export otel", err, stderr, opts.JSON)
			return 5
		}
		spanCount := otelSpanCount(payload)
		if opts.JSON {
			writeJSON(stdout, map[string]any{
				"command":     "export otel",
				"status":      "success",
				"format":      "otel",
				"spans":       spanCount,
				"output_path": exportOpts.OutputPath,
			})
			return 0
		}
		fmt.Fprintf(stdout, "OTel traces written to %s\n", exportOpts.OutputPath)
		fmt.Fprintf(stdout, "Spans: %d\n", spanCount)
		return 0
	}
	if opts.JSON {
		writeJSON(stdout, map[string]any{
			"command": "export otel",
			"status":  "success",
			"format":  "otel",
			"spans":   otelSpanCount(payload),
			"payload": payload,
		})
		return 0
	}
	if err := telemetry.WriteOTLPJSON(stdout, payload); err != nil {
		printError("export otel", err, stderr, opts.JSON)
		return 5
	}
	return 0
}

type exportOptions struct {
	Format     string
	OutputPath string
}

func parseExportArgs(args []string) (exportOptions, error) {
	opts := exportOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--output":
			i++
			if i >= len(args) {
				return exportOptions{}, fmt.Errorf("--output requires a value")
			}
			opts.OutputPath = args[i]
		case strings.HasPrefix(arg, "--output="):
			opts.OutputPath = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "--"):
			return exportOptions{}, fmt.Errorf("unknown export flag: %s", arg)
		default:
			if opts.Format != "" {
				return exportOptions{}, fmt.Errorf("export accepts one format")
			}
			opts.Format = arg
		}
	}
	if opts.Format == "" {
		return exportOptions{}, fmt.Errorf("export requires a format")
	}
	return opts, nil
}

func writeOpenLineageOutput(path string, events []lineage.RunEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return lineage.WriteOpenLineageNDJSON(file, events)
}

func writeOTelOutput(path string, payload telemetry.TracesData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return telemetry.WriteOTLPJSON(file, payload)
}

func otelSpanCount(payload telemetry.TracesData) int {
	var count int
	for _, resourceSpans := range payload.ResourceSpans {
		for _, scopeSpans := range resourceSpans.ScopeSpans {
			count += len(scopeSpans.Spans)
		}
	}
	return count
}

func runInit(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := rejectSelect("init", opts); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	initOpts, err := parseInitArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	destination := opts.ProjectDir
	if destination == "" {
		destination = initOpts.ProjectName
	}
	result, err := templates.CreateProject(templates.Options{
		ProjectName: initOpts.ProjectName,
		Destination: destination,
		Template:    initOpts.Template,
		Force:       initOpts.Force,
	})
	if err != nil {
		printError("init", err, stderr, opts.JSON)
		return 2
	}
	if opts.JSON {
		writeJSON(stdout, map[string]any{"command": "init", "status": "success", "project": result})
		return 0
	}
	fmt.Fprintf(stdout, "Initialized %s project at %s\n", result.Template, result.ProjectDir)
	fmt.Fprintf(stdout, "Files: %d\n", len(result.Files))
	if result.Template == "support" || result.Template == "incident" {
		fmt.Fprintln(stdout, "Demo runners: configured as demo.*; replace runner commands in fs_project.yml for real provider execution")
	}
	return 0
}

type initOptions struct {
	ProjectName string
	Template    string
	Force       bool
}

func parseInitArgs(args []string) (initOptions, error) {
	opts := initOptions{ProjectName: "fbt_project", Template: "blank"}
	seenProjectName := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--template":
			i++
			if i >= len(args) {
				return initOptions{}, fmt.Errorf("--template requires a value")
			}
			opts.Template = args[i]
		case strings.HasPrefix(arg, "--template="):
			opts.Template = strings.TrimPrefix(arg, "--template=")
		case arg == "--force":
			opts.Force = true
		case strings.HasPrefix(arg, "--"):
			return initOptions{}, fmt.Errorf("unknown init flag: %s", arg)
		default:
			if seenProjectName {
				return initOptions{}, fmt.Errorf("init accepts at most one project name")
			}
			opts.ProjectName = arg
			seenProjectName = true
		}
	}
	return opts, nil
}

func runBuild(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := expectNoArgs("build", args); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	result, err := buildmgr.RunBuild(context.Background(), buildmgr.Options{
		ProjectDir: opts.ProjectDir,
		StateDir:   opts.StateDir,
		Select:     opts.Select,
		FBTVersion: versioninfo.Version,
	})
	if err != nil {
		printError("build", err, stderr, opts.JSON)
		if errors.Is(err, runnermgr.ErrCapabilityIncompatible) {
			return 6
		}
		if isSelectionError(err) {
			return 2
		}
		return 1
	}
	if opts.JSON {
		writeJSON(stdout, map[string]any{"command": "build", "status": "success", "summary": result.Plan.Summary, "runs": result.Runs})
		if result.Plan.Summary.Blocked > 0 {
			return 3
		}
		return 0
	}
	fmt.Fprintf(stdout, "Build: %d selected, %d run, %d skipped, %d blocked\n", result.Plan.Summary.Selected, result.Plan.Summary.Run, result.Plan.Summary.Skipped, result.Plan.Summary.Blocked)
	for _, run := range result.Runs {
		fmt.Fprintf(stdout, "%s %s\n", run.Status, run.TransformID)
		for _, version := range run.CommittedVersions {
			fmt.Fprintf(stdout, "  committed: %s\n", version)
		}
	}
	for _, node := range result.Plan.Nodes {
		if node.Action == planner.ActionRun {
			continue
		}
		printPlanNode(stdout, node)
	}
	if result.Plan.Summary.Blocked > 0 {
		return 3
	}
	return 0
}

func isSelectionError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "selector matched no transforms") || strings.Contains(message, "unknown selector")
}

type doctorCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

func runDoctor(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := rejectSelect("doctor", opts); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	if err := expectNoArgs("doctor", args); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	checks := []doctorCheck{
		{Name: "project_config", Status: "ok", Code: "PROJECT_CONFIG_OK", Severity: "info", Message: "project config parsed"},
	}
	checks = append(checks, stateDoctorChecks(ctx.Store)...)
	checks = append(checks, runnerDoctorChecks(ctx.ParseResult.ProjectDir, ctx.ParseResult.Config, ctx.Manifest)...)
	status := "ok"
	code := 0
	if doctorHasErrors(checks) {
		status = "error"
		code = 6
	}
	if opts.JSON {
		writeJSON(stdout, map[string]any{"command": "doctor", "status": status, "project_dir": ctx.ParseResult.ProjectDir, "state_dir": ctx.Store.Dir, "checks": checks})
		return code
	}
	fmt.Fprintf(stdout, "Doctor: %s\n", status)
	fmt.Fprintf(stdout, "Project: %s\n", ctx.ParseResult.ProjectDir)
	fmt.Fprintf(stdout, "State: %s\n", ctx.Store.Dir)
	for _, check := range checks {
		fmt.Fprintf(stdout, "%s %s: %s\n", check.Status, check.Code, check.Message)
	}
	return code
}

func stateDoctorChecks(store state.Store) []doctorCheck {
	var checks []doctorCheck
	lock, err := store.AcquireLock("doctor", time.Minute)
	if err != nil {
		return append(checks, doctorError("state_lock", "STATE_LOCK_ERROR", err.Error()))
	}
	if err := lock.Release(); err != nil {
		checks = append(checks, doctorError("state_lock", "STATE_LOCK_RELEASE_ERROR", err.Error()))
	} else {
		checks = append(checks, doctorOK("state_lock", "STATE_LOCK_OK", "state lock can be acquired and released"))
	}
	tmp, err := os.CreateTemp(store.Dir, ".doctor-*.tmp")
	if err != nil {
		return append(checks, doctorError("state_writable", "STATE_WRITABLE_ERROR", err.Error()))
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		checks = append(checks, doctorError("state_writable", "STATE_WRITABLE_ERROR", err.Error()))
	} else if err := os.Remove(tmpPath); err != nil {
		checks = append(checks, doctorError("state_writable", "STATE_CLEANUP_ERROR", err.Error()))
	} else {
		checks = append(checks, doctorOK("state_writable", "STATE_WRITABLE_OK", "state directory is writable"))
	}
	return checks
}

func runnerDoctorChecks(projectDir string, cfg config.ProjectConfig, m manifest.Manifest) []doctorCheck {
	discovery := runnermgr.NewDiscovery(projectDir, cfg)
	resolved, err := discovery.List()
	if err != nil {
		return []doctorCheck{doctorError("runner_discovery", "RUNNER_DISCOVERY_ERROR", err.Error())}
	}
	if len(resolved) == 0 {
		return []doctorCheck{doctorOK("runner_discovery", "RUNNER_DISCOVERY_EMPTY", "no runners configured")}
	}
	var checks []doctorCheck
	for _, runner := range resolved {
		diagnostics := runnermgr.Diagnose(runner)
		if runnermgr.HasErrors(diagnostics) {
			for _, diagnostic := range diagnostics {
				status := "ok"
				if diagnostic.Severity == "error" {
					status = "error"
				}
				checks = append(checks, doctorCheck{Name: "runner." + runner.Name, Status: status, Code: diagnostic.Code, Severity: diagnostic.Severity, Message: diagnostic.Message})
			}
			continue
		}
		for _, diagnostic := range diagnostics {
			checks = append(checks, doctorCheck{Name: "runner." + runner.Name, Status: "ok", Code: diagnostic.Code, Severity: diagnostic.Severity, Message: diagnostic.Message})
		}
		checks = append(checks, runnerProtocolDoctorChecks(runner, capabilityRequirementsForRunner(m, runner.Name))...)
	}
	return checks
}

func runnerProtocolDoctorChecks(resolved runnermgr.Resolved, requirements []runnermgr.CapabilityRequirement) []doctorCheck {
	var checks []doctorCheck
	for _, diagnostic := range runnerProtocolDiagnostics(resolved, requirements) {
		status := "ok"
		if diagnostic.Severity == "error" {
			status = "error"
		}
		checks = append(checks, doctorCheck{Name: "runner." + resolved.Name, Status: status, Code: diagnostic.Code, Severity: diagnostic.Severity, Message: diagnostic.Message})
	}
	return checks
}

func runnerProtocolDiagnostics(resolved runnermgr.Resolved, requirements []runnermgr.CapabilityRequirement) []runnermgr.Diagnostic {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := runnermgr.StartProtocolClient(ctx, resolved)
	if err != nil {
		return []runnermgr.Diagnostic{{Severity: "error", Code: "RUNNER_PROTOCOL_START_ERROR", Message: err.Error()}}
	}
	defer client.Close()
	initResult, err := client.Initialize(ctx, protocol.InitializeParams{
		Core: map[string]string{"name": "fbt-core", "version": versioninfo.Version},
		Protocol: map[string]any{
			"versions": []string{"0.1"},
			"framing":  "jsonl",
		},
		CapabilityRequest: []string{"run_transform", "stream_events", "output_candidates", "cancellation"},
	})
	if err != nil {
		return []runnermgr.Diagnostic{{Severity: "error", Code: "RUNNER_PROTOCOL_INIT_ERROR", Message: err.Error()}}
	}
	diagnostics := []runnermgr.Diagnostic{{Severity: "info", Code: "RUNNER_PROTOCOL_OK", Message: "runner protocol initialize succeeded"}}
	diagnostics = append(diagnostics, runnermgr.ValidateCapabilities(initResult, requirements)...)
	return diagnostics
}

func capabilityRequirementsForRunner(m manifest.Manifest, runnerName string) []runnermgr.CapabilityRequirement {
	var requirements []runnermgr.CapabilityRequirement
	for transformID, transform := range m.Transforms {
		runnerResource, ok := m.Runners[transform.Runner]
		if !ok || runnerResource.Name != runnerName {
			continue
		}
		artifactTypes := make([]string, 0, len(transform.Outputs))
		for _, output := range transform.Outputs {
			artifactTypes = append(artifactTypes, output.ArtifactType)
		}
		requirements = append(requirements, runnermgr.CapabilityRequirement{
			TransformID:             transformID,
			TransformType:           transform.TransformType,
			ArtifactTypes:           artifactTypes,
			RequireOutputCandidates: true,
		})
	}
	return requirements
}

func doctorOK(name, code, message string) doctorCheck {
	return doctorCheck{Name: name, Status: "ok", Code: code, Severity: "info", Message: message}
}

func doctorError(name, code, message string) doctorCheck {
	return doctorCheck{Name: name, Status: "error", Code: code, Severity: "error", Message: message}
}

func doctorHasErrors(checks []doctorCheck) bool {
	for _, check := range checks {
		if check.Status == "error" || check.Severity == "error" {
			return true
		}
	}
	return false
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "fbt - file build tool")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  fbt <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Primary commands:")
	for _, command := range primaryCommands {
		fmt.Fprintf(w, "  %-10s %s\n", command.Name, command.Description)
	}
	fmt.Fprintln(w, "  version    Print version")
	fmt.Fprintln(w, "  help       Show this help")
}

func parseOptions(args []string) (options, []string, error) {
	var opts options
	var commandArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			opts.JSON = true
		case arg == "--project-dir":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--project-dir requires a value")
			}
			opts.ProjectDir = args[i]
		case strings.HasPrefix(arg, "--project-dir="):
			opts.ProjectDir = strings.TrimPrefix(arg, "--project-dir=")
		case arg == "--state-dir":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--state-dir requires a value")
			}
			opts.StateDir = args[i]
		case strings.HasPrefix(arg, "--state-dir="):
			opts.StateDir = strings.TrimPrefix(arg, "--state-dir=")
		case arg == "--select":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--select requires a value")
			}
			opts.Select = args[i]
		case strings.HasPrefix(arg, "--select="):
			opts.Select = strings.TrimPrefix(arg, "--select=")
		default:
			commandArgs = append(commandArgs, arg)
		}
	}
	return opts, commandArgs, nil
}

type projectContext struct {
	ParseResult parser.Result
	Manifest    manifest.Manifest
	Store       state.Store
}

func loadProject(opts options) (projectContext, error) {
	parseResult, err := parser.ParseProject(parser.Options{ProjectDir: opts.ProjectDir})
	if err != nil {
		return projectContext{ParseResult: parseResult}, err
	}
	m, err := manifest.Build(parseResult, manifest.BuildOptions{FBTVersion: versioninfo.Version})
	if err != nil {
		return projectContext{ParseResult: parseResult}, err
	}
	stateDir := opts.StateDir
	if stateDir == "" {
		stateDir = filepath.Join(parseResult.ProjectDir, parseResult.Config.State.Path)
	}
	return projectContext{ParseResult: parseResult, Manifest: m, Store: state.Open(stateDir)}, nil
}

func runPlan(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := expectNoArgs("plan", args); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	var previous *manifest.Manifest
	if prev, err := ctx.Store.ReadManifest(); err == nil {
		previous = &prev
	} else if !errors.Is(err, os.ErrNotExist) {
		printError("plan", err, stderr, opts.JSON)
		return 5
	}
	snapshot, err := ctx.Store.ReadState()
	if err != nil {
		printError("plan", err, stderr, opts.JSON)
		return 5
	}
	selected, err := selectedTransformIDs(ctx.Manifest, opts.Select)
	if err != nil {
		printError("plan", err, stderr, opts.JSON)
		return 2
	}
	plan := planner.Build(planner.Inputs{Manifest: ctx.Manifest, PreviousManifest: previous, State: snapshot, Selected: selected})
	if opts.JSON {
		writeJSON(stdout, map[string]any{"command": "plan", "status": "success", "summary": plan.Summary, "nodes": plan.Nodes})
		return 0
	}
	fmt.Fprintf(stdout, "Plan: %d selected, %d run, %d skipped, %d blocked\n", plan.Summary.Selected, plan.Summary.Run, plan.Summary.Skipped, plan.Summary.Blocked)
	for _, node := range plan.Nodes {
		printPlanNode(stdout, node)
	}
	return 0
}

func printPlanNode(stdout io.Writer, node planner.Node) {
	fmt.Fprintf(stdout, "%s %s\n", node.Action, node.TransformID)
	for _, reason := range node.DirtyReasons {
		fmt.Fprintf(stdout, "  reason: %s\n", reason)
	}
	for _, reason := range node.BlockedReasons {
		fmt.Fprintf(stdout, "  blocked: %s\n", reason)
	}
	for _, step := range node.NextSteps {
		fmt.Fprintf(stdout, "  next: %s\n", step)
	}
}

func runArtifact(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := rejectSelect("artifact", opts); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	subcommand := "ls"
	if len(args) > 0 {
		subcommand = args[0]
	}
	versions, err := ctx.Store.ReadArtifactVersions()
	if err != nil {
		printError("artifact", err, stderr, opts.JSON)
		return 5
	}
	switch subcommand {
	case "ls":
		if err := expectAtMostArgs("artifact ls", args, 1); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		ids := artifactIDs(versions)
		if opts.JSON {
			writeJSON(stdout, map[string]any{"command": "artifact ls", "status": "success", "artifacts": ids})
			return 0
		}
		for _, id := range ids {
			fmt.Fprintln(stdout, id)
		}
		return 0
	case "show":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "Error: artifact show requires 1 argument(s)")
			return 2
		}
		if err := expectArgs("artifact show", args, 2); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		return runArtifactShow(ctx, versions, args[1], opts.JSON, stdout, stderr)
	case "explain":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "Error: artifact explain requires 1 argument(s)")
			return 2
		}
		if err := expectArgs("artifact explain", args, 2); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		return runArtifactExplain(ctx, args[1], opts.JSON, stdout, stderr)
	case "path":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "Error: artifact path requires 1 argument(s)")
			return 2
		}
		if err := expectArgs("artifact path", args, 2); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		return runArtifactPath(ctx, versions, args[1], opts.JSON, stdout, stderr)
	case "history":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "Error: artifact history requires 1 argument(s)")
			return 2
		}
		if err := expectArgs("artifact history", args, 2); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		return runArtifactHistory(ctx, versions, args[1], opts.JSON, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown artifact command: %s\n", subcommand)
		return 2
	}
}

type artifactRecord struct {
	VersionID           string              `json:"version_id,omitempty"`
	ArtifactID          string              `json:"artifact_id"`
	LogicalPath         string              `json:"logical_path,omitempty"`
	StoragePath         string              `json:"storage_path,omitempty"`
	AbsoluteLogicalPath string              `json:"absolute_logical_path,omitempty"`
	AbsoluteStoragePath string              `json:"absolute_storage_path,omitempty"`
	Digest              string              `json:"digest,omitempty"`
	ArtifactType        string              `json:"artifact_type,omitempty"`
	GeneratedBy         string              `json:"generated_by,omitempty"`
	Runner              string              `json:"runner,omitempty"`
	Model               map[string]any      `json:"model,omitempty"`
	Confidence          string              `json:"confidence,omitempty"`
	Current             bool                `json:"current"`
	CreatedAt           string              `json:"created_at,omitempty"`
	CommittedAt         string              `json:"committed_at,omitempty"`
	Materials           []state.Material    `json:"materials,omitempty"`
	SemanticDescriptor  map[string]any      `json:"semantic_descriptor,omitempty"`
	Descriptor          artifact.Descriptor `json:"descriptor,omitempty"`
}

func runArtifactPath(ctx projectContext, versions state.ArtifactVersionsIndex, target string, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	snapshot, err := ctx.Store.ReadState()
	if err != nil {
		printError("artifact path", err, stderr, jsonOutput)
		return 5
	}
	if version, ok := findVersion(snapshot, versions, target); ok {
		record := buildArtifactRecord(ctx, snapshot, version)
		if jsonOutput {
			writeJSON(stdout, map[string]any{"command": "artifact path", "status": "success", "artifact": record})
			return 0
		}
		printArtifactPath(stdout, record)
		return 0
	}
	record, ok := declaredArtifactRecord(ctx, target)
	if !ok {
		fmt.Fprintf(stderr, "Error: artifact not found: %s\n", target)
		return 2
	}
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": "artifact path", "status": "success", "artifact": record})
		return 0
	}
	printArtifactPath(stdout, record)
	return 0
}

func runArtifactShow(ctx projectContext, versions state.ArtifactVersionsIndex, target string, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	snapshot, err := ctx.Store.ReadState()
	if err != nil {
		printError("artifact show", err, stderr, jsonOutput)
		return 5
	}
	version, ok := findVersion(snapshot, versions, target)
	if !ok {
		fmt.Fprintf(stderr, "Error: artifact not found: %s\n", target)
		return 2
	}
	record := buildArtifactRecord(ctx, snapshot, version)
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": "artifact show", "status": "success", "artifact": record})
		return 0
	}
	printArtifactRecord(stdout, record)
	return 0
}

func runArtifactHistory(ctx projectContext, versions state.ArtifactVersionsIndex, target string, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	snapshot, err := ctx.Store.ReadState()
	if err != nil {
		printError("artifact history", err, stderr, jsonOutput)
		return 5
	}
	matches := matchingVersions(versions, target)
	if len(matches) == 0 {
		fmt.Fprintf(stderr, "Error: artifact not found: %s\n", target)
		return 2
	}
	sortArtifactVersions(matches)
	records := make([]artifactRecord, 0, len(matches))
	for _, version := range matches {
		records = append(records, buildArtifactRecord(ctx, snapshot, version))
	}
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": "artifact history", "status": "success", "artifacts": records})
		return 0
	}
	for _, record := range records {
		printArtifactRecord(stdout, record)
	}
	return 0
}

type artifactExplanation struct {
	ArtifactID     string                     `json:"artifact_id"`
	TransformID    string                     `json:"transform_id"`
	TransformName  string                     `json:"transform_name"`
	Action         planner.Action             `json:"action"`
	Inputs         []manifest.TransformInput  `json:"inputs,omitempty"`
	Outputs        []manifest.TransformOutput `json:"outputs,omitempty"`
	DirtyReasons   []string                   `json:"dirty_reasons,omitempty"`
	BlockedReasons []string                   `json:"blocked_reasons,omitempty"`
	NextSteps      []string                   `json:"next_steps,omitempty"`
	Current        *state.ArtifactPointer     `json:"current,omitempty"`
	PreviousRun    *state.LatestRun           `json:"previous_run,omitempty"`
}

func runArtifactExplain(ctx projectContext, target string, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	artifactID, ok := resolveArtifactID(ctx.Manifest, target)
	if !ok {
		fmt.Fprintf(stderr, "Error: artifact not found: %s\n", target)
		return 2
	}
	transform, ok := producerTransform(ctx.Manifest, artifactID)
	if !ok {
		fmt.Fprintf(stderr, "Error: producer transform not found for %s\n", artifactID)
		return 2
	}
	var previous *manifest.Manifest
	if prev, err := ctx.Store.ReadManifest(); err == nil {
		previous = &prev
	} else if !errors.Is(err, os.ErrNotExist) {
		printError("artifact explain", err, stderr, jsonOutput)
		return 5
	}
	snapshot, err := ctx.Store.ReadState()
	if err != nil {
		printError("artifact explain", err, stderr, jsonOutput)
		return 5
	}
	plan := planner.Build(planner.Inputs{
		Manifest:         ctx.Manifest,
		PreviousManifest: previous,
		State:            snapshot,
		Selected:         map[string]struct{}{transform.UniqueID: {}},
	})
	if len(plan.Nodes) == 0 {
		fmt.Fprintf(stderr, "Error: plan node not found for %s\n", transform.UniqueID)
		return 2
	}
	node := plan.Nodes[0]
	explanation := artifactExplanation{
		ArtifactID:     artifactID,
		TransformID:    transform.UniqueID,
		TransformName:  transform.Name,
		Action:         node.Action,
		Inputs:         transform.Inputs,
		Outputs:        transform.Outputs,
		DirtyReasons:   node.DirtyReasons,
		BlockedReasons: node.BlockedReasons,
		NextSteps:      node.NextSteps,
	}
	if pointer, ok := snapshot.CurrentArtifacts[artifactID]; ok {
		explanation.Current = &pointer
	}
	if latest, ok := snapshot.LatestRuns[transform.UniqueID]; ok {
		explanation.PreviousRun = &latest
	}
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": "artifact explain", "status": "success", "explanation": explanation})
		return 0
	}
	printArtifactExplanation(stdout, explanation)
	return 0
}

func selectedTransformIDs(m manifest.Manifest, expr string) (map[string]struct{}, error) {
	if expr == "" {
		return nil, nil
	}
	var ids []string
	var err error
	switch {
	case strings.HasPrefix(expr, "selector:"):
		name := strings.TrimPrefix(expr, "selector:")
		definition, ok := m.Selectors[name]
		if !ok {
			return nil, fmt.Errorf("unknown selector: %s", name)
		}
		ids, err = graph.SelectDefinition(m, definition)
	case strings.HasPrefix(expr, "tag:"):
		ids, err = graph.Select(m, graph.Selector{Method: "tag", Value: strings.TrimPrefix(expr, "tag:")})
	case strings.HasPrefix(expr, "path:"):
		ids, err = graph.Select(m, graph.Selector{Method: "path", Value: strings.TrimPrefix(expr, "path:")})
	case strings.HasPrefix(expr, "resource_type:"):
		ids, err = graph.Select(m, graph.Selector{Method: "resource_type", Value: strings.TrimPrefix(expr, "resource_type:")})
	default:
		ids, err = graph.Select(m, graph.Selector{Method: "name", Value: strings.Trim(expr, "+")})
	}
	if err != nil {
		return nil, err
	}
	selected := map[string]struct{}{}
	for _, id := range ids {
		if _, ok := m.Transforms[id]; ok {
			selected[id] = struct{}{}
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("selector matched no transforms: %s", expr)
	}
	return selected, nil
}

func buildArtifactRecord(ctx projectContext, snapshot state.Snapshot, version state.ArtifactVersion) artifactRecord {
	record := artifactRecord{
		VersionID:           version.VersionID,
		ArtifactID:          version.ArtifactID,
		LogicalPath:         version.LogicalPath,
		StoragePath:         version.StoragePath,
		Digest:              version.Descriptor.Digest,
		ArtifactType:        version.Descriptor.ArtifactType,
		GeneratedBy:         version.GeneratedBy,
		Confidence:          version.Confidence,
		CreatedAt:           version.CreatedAt,
		CommittedAt:         version.CommittedAt,
		Materials:           version.Materials,
		SemanticDescriptor:  version.SemanticDescriptor,
		Descriptor:          version.Descriptor,
		AbsoluteLogicalPath: absoluteProjectPath(ctx.ParseResult.ProjectDir, version.LogicalPath),
		AbsoluteStoragePath: absoluteProjectPath(ctx.ParseResult.ProjectDir, version.StoragePath),
	}
	if pointer, ok := snapshot.CurrentArtifacts[version.ArtifactID]; ok && pointer.CurrentVersionID == version.VersionID {
		record.Current = true
		if pointer.Confidence != "" {
			record.Confidence = pointer.Confidence
		}
	}
	if transform, ok := producerTransform(ctx.Manifest, version.ArtifactID); ok {
		record.Runner = transform.Runner
		record.Model = transform.Model
	}
	return record
}

func declaredArtifactRecord(ctx projectContext, target string) (artifactRecord, bool) {
	artifactID, ok := resolveArtifactID(ctx.Manifest, target)
	if !ok {
		return artifactRecord{}, false
	}
	record := artifactRecord{ArtifactID: artifactID}
	if transform, ok := producerTransform(ctx.Manifest, artifactID); ok {
		record.Runner = transform.Runner
		record.Model = transform.Model
		for _, output := range transform.Outputs {
			if output.UniqueID == artifactID {
				record.LogicalPath = output.Name
				if artifactResource, ok := ctx.Manifest.Artifacts[artifactID]; ok && artifactResource.LogicalPath != "" {
					record.LogicalPath = artifactResource.LogicalPath
				}
				record.AbsoluteLogicalPath = absoluteProjectPath(ctx.ParseResult.ProjectDir, record.LogicalPath)
				break
			}
		}
	}
	if artifactResource, ok := ctx.Manifest.Artifacts[artifactID]; ok && record.LogicalPath == "" {
		record.LogicalPath = artifactResource.LogicalPath
		record.AbsoluteLogicalPath = absoluteProjectPath(ctx.ParseResult.ProjectDir, record.LogicalPath)
	}
	return record, true
}

func absoluteProjectPath(root, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, filepath.FromSlash(path))
}

func printArtifactPath(stdout io.Writer, record artifactRecord) {
	fmt.Fprintf(stdout, "logical_path: %s\n", record.LogicalPath)
	if record.StoragePath != "" {
		fmt.Fprintf(stdout, "storage_path: %s\n", record.StoragePath)
	}
	if record.VersionID != "" {
		fmt.Fprintf(stdout, "version: %s\n", record.VersionID)
	} else {
		fmt.Fprintln(stdout, "version: none")
	}
}

func printArtifactRecord(stdout io.Writer, record artifactRecord) {
	if record.VersionID != "" {
		fmt.Fprintf(stdout, "%s\n", record.VersionID)
	} else {
		fmt.Fprintf(stdout, "%s\n", record.ArtifactID)
	}
	fmt.Fprintf(stdout, "  artifact: %s\n", record.ArtifactID)
	if record.Current {
		fmt.Fprintln(stdout, "  current: true")
	}
	if record.Digest != "" {
		fmt.Fprintf(stdout, "  digest: %s\n", record.Digest)
	}
	if record.ArtifactType != "" {
		fmt.Fprintf(stdout, "  type: %s\n", record.ArtifactType)
	}
	if record.LogicalPath != "" {
		fmt.Fprintf(stdout, "  logical_path: %s\n", record.LogicalPath)
	}
	if record.StoragePath != "" {
		fmt.Fprintf(stdout, "  storage_path: %s\n", record.StoragePath)
	}
	if record.Runner != "" {
		fmt.Fprintf(stdout, "  runner: %s\n", record.Runner)
	}
	if len(record.Model) > 0 {
		fmt.Fprintf(stdout, "  model: %s\n", compactJSON(record.Model))
	}
	if record.GeneratedBy != "" {
		fmt.Fprintf(stdout, "  generated_by: %s\n", record.GeneratedBy)
	}
	if len(record.SemanticDescriptor) > 0 {
		fmt.Fprintf(stdout, "  semantic_descriptor: %s\n", compactJSON(record.SemanticDescriptor))
	}
	if record.Confidence != "" {
		fmt.Fprintf(stdout, "  confidence: %s\n", record.Confidence)
	}
	if record.CommittedAt != "" {
		fmt.Fprintf(stdout, "  committed_at: %s\n", record.CommittedAt)
	}
	for _, material := range record.Materials {
		fmt.Fprintf(stdout, "  material: %s", material.ResourceID)
		if material.ArtifactVersion != "" {
			fmt.Fprintf(stdout, " %s", material.ArtifactVersion)
		}
		if material.Digest != "" {
			fmt.Fprintf(stdout, " %s", material.Digest)
		}
		fmt.Fprintln(stdout)
	}
}

func compactJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func printArtifactExplanation(stdout io.Writer, explanation artifactExplanation) {
	fmt.Fprintf(stdout, "%s\n", explanation.ArtifactID)
	fmt.Fprintf(stdout, "  transform: %s\n", explanation.TransformID)
	fmt.Fprintf(stdout, "  action: %s\n", explanation.Action)
	for _, input := range explanation.Inputs {
		fmt.Fprintf(stdout, "  input: %s %s\n", input.Kind, input.UniqueID)
	}
	for _, output := range explanation.Outputs {
		fmt.Fprintf(stdout, "  output: %s\n", output.UniqueID)
	}
	if explanation.Current != nil {
		fmt.Fprintf(stdout, "  current_version: %s\n", explanation.Current.CurrentVersionID)
		fmt.Fprintf(stdout, "  confidence: %s\n", explanation.Current.Confidence)
	} else {
		fmt.Fprintln(stdout, "  current_version: none")
	}
	if explanation.PreviousRun != nil {
		fmt.Fprintf(stdout, "  previous_run: %s %s\n", explanation.PreviousRun.LatestRunID, explanation.PreviousRun.LatestStatus)
	} else {
		fmt.Fprintln(stdout, "  previous_run: none")
	}
	for _, reason := range explanation.DirtyReasons {
		fmt.Fprintf(stdout, "  reason: %s\n", reason)
	}
	for _, reason := range explanation.BlockedReasons {
		fmt.Fprintf(stdout, "  blocked: %s\n", reason)
	}
	for _, step := range explanation.NextSteps {
		fmt.Fprintf(stdout, "  next: %s\n", step)
	}
}

func resolveArtifactID(m manifest.Manifest, target string) (string, bool) {
	if _, ok := m.Artifacts[target]; ok {
		return target, true
	}
	for id, artifact := range m.Artifacts {
		if artifact.Name == target || strings.HasSuffix(id, "."+target) {
			return id, true
		}
	}
	for _, transform := range m.Transforms {
		for _, output := range transform.Outputs {
			if output.Name == target || output.UniqueID == target || strings.HasSuffix(output.UniqueID, "."+target) {
				return output.UniqueID, true
			}
		}
	}
	return "", false
}

func matchingVersions(index state.ArtifactVersionsIndex, target string) []state.ArtifactVersion {
	var matches []state.ArtifactVersion
	for _, version := range index.ArtifactVersions {
		if version.VersionID == target || version.ArtifactID == target || strings.HasSuffix(version.ArtifactID, "."+target) {
			matches = append(matches, version)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].VersionID < matches[j].VersionID
	})
	return matches
}

func sortArtifactVersions(versions []state.ArtifactVersion) {
	sort.Slice(versions, func(i, j int) bool {
		left := versions[i].CommittedAt
		if left == "" {
			left = versions[i].CreatedAt
		}
		right := versions[j].CommittedAt
		if right == "" {
			right = versions[j].CreatedAt
		}
		if left != right {
			return left < right
		}
		return versions[i].VersionID < versions[j].VersionID
	})
}

func findVersion(snapshot state.Snapshot, index state.ArtifactVersionsIndex, target string) (state.ArtifactVersion, bool) {
	if version, ok := index.ArtifactVersions[target]; ok {
		return version, true
	}
	if id, pointer, ok := findPointer(snapshot, target); ok {
		_ = id
		version, exists := index.ArtifactVersions[pointer.CurrentVersionID]
		return version, exists
	}
	matches := matchingVersions(index, target)
	if len(matches) == 0 {
		return state.ArtifactVersion{}, false
	}
	return matches[len(matches)-1], true
}

func resolveAgainst(store state.Store, snapshot state.Snapshot, index state.ArtifactVersionsIndex, current state.ArtifactVersion, against string) (state.ArtifactVersion, error) {
	switch against {
	case "", "previous":
		version, ok := previousVersion(index, current)
		if !ok {
			return state.ArtifactVersion{}, fmt.Errorf("previous artifact version not found for %s", current.ArtifactID)
		}
		return version, nil
	default:
		version, ok := findVersion(snapshot, index, against)
		if !ok {
			return state.ArtifactVersion{}, fmt.Errorf("artifact version not found: %s", against)
		}
		return version, nil
	}
}

func previousVersion(index state.ArtifactVersionsIndex, current state.ArtifactVersion) (state.ArtifactVersion, bool) {
	matches := matchingVersions(index, current.ArtifactID)
	var previous []state.ArtifactVersion
	for _, version := range matches {
		if version.VersionID != current.VersionID {
			previous = append(previous, version)
		}
	}
	if len(previous) == 0 {
		return state.ArtifactVersion{}, false
	}
	return previous[len(previous)-1], true
}

func producerTransform(m manifest.Manifest, artifactID string) (manifest.TransformResource, bool) {
	for _, transform := range m.Transforms {
		for _, output := range transform.Outputs {
			if output.UniqueID == artifactID {
				return transform, true
			}
		}
	}
	return manifest.TransformResource{}, false
}

func shortResourceName(id string) string {
	if index := strings.LastIndex(id, "."); index >= 0 && index+1 < len(id) {
		return id[index+1:]
	}
	return id
}

func artifactIDs(index state.ArtifactVersionsIndex) []string {
	seen := map[string]struct{}{}
	for _, version := range index.ArtifactVersions {
		seen[version.ArtifactID] = struct{}{}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func findPointer(snapshot state.Snapshot, target string) (string, state.ArtifactPointer, bool) {
	for id, pointer := range snapshot.CurrentArtifacts {
		if id == target || strings.HasSuffix(id, "."+target) {
			return id, pointer, true
		}
	}
	return "", state.ArtifactPointer{}, false
}

func printParseError(err error, diagnostics []parser.Diagnostic, stderr io.Writer, jsonOutput bool) {
	if jsonOutput {
		writeJSON(stderr, map[string]any{"status": "error", "error": err.Error(), "diagnostics": diagnostics})
		return
	}
	fmt.Fprintf(stderr, "Error: %v\n", err)
	for _, diagnostic := range diagnostics {
		location := diagnostic.File
		if diagnostic.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, diagnostic.Line)
		}
		if location != "" {
			fmt.Fprintf(stderr, "  %s: %s (%s)\n", diagnostic.Code, diagnostic.Message, location)
		} else {
			fmt.Fprintf(stderr, "  %s: %s\n", diagnostic.Code, diagnostic.Message)
		}
		if diagnostic.Hint != "" {
			fmt.Fprintf(stderr, "    hint: %s\n", diagnostic.Hint)
		}
	}
}

func printError(command string, err error, stderr io.Writer, jsonOutput bool) {
	if jsonOutput {
		writeJSON(stderr, map[string]any{"command": command, "status": "error", "error": err.Error()})
		return
	}
	fmt.Fprintf(stderr, "Error: %v\n", err)
}

func writeJSON(w io.Writer, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintf(w, `{"status":"error","error":%q}`+"\n", err.Error())
		return
	}
	fmt.Fprintf(w, "%s\n", data)
}
