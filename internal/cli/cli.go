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
	"text/tabwriter"
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
	"github.com/spf13/cobra"
)

type options struct {
	ProjectDir string
	StateDir   string
	JSON       bool
	Select     string
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	root := newRootCommand(stdout, stderr)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		var exitErr exitCodeError
		if errors.As(err, &exitErr) {
			return exitErr.code
		}
		printCLIError(stderr, err.Error())
		return 2
	}
	return 0
}

type exitCodeError struct {
	code int
}

func (e exitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

func exitCode(code int) error {
	if code == 0 {
		return nil
	}
	return exitCodeError{code: code}
}

func newRootCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	opts := options{}
	versionFlag := false
	root := &cobra.Command{
		Use:   "fbt",
		Short: "fbt - file build tool",
		Long: "fbt builds versioned filesystem artifacts from declared source files through external runners.\n\n" +
			"A runner is an external command that speaks the fbt runner protocol; it can\n" +
			"wrap OpenAI, Claude Code, Codex, Gemini, a script, or an internal service.\n\n" +
			"Typical flow:\n" +
			"  fbt doctor\n" +
			"  fbt plan --select TARGET\n" +
			"  fbt build --select TARGET\n" +
			"  fbt artifact show TARGET\n\n" +
			"Use --json for automation. The human output is optimized for scanning local project state.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown command: %s", args[0])
			}
			if versionFlag {
				return exitCode(runVersion(opts, stdout, stderr))
			}
			return cmd.Help()
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.CompletionOptions.DisableDefaultCmd = true
	root.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version")
	root.PersistentFlags().StringVar(&opts.ProjectDir, "project-dir", "", "Directory containing fs_project.yml")
	root.PersistentFlags().StringVar(&opts.StateDir, "state-dir", "", "Override .fbt/state; immutable artifact storage stays under .fbt/artifacts")
	root.PersistentFlags().StringVar(&opts.Select, "select", "", "Select transforms for plan/build; rejected by inspection commands")
	root.PersistentFlags().BoolVar(&opts.JSON, "json", false, "Print machine-readable JSON")

	root.AddCommand(newVersionCommand(&opts, stdout, stderr))
	root.AddCommand(newInitCommand(&opts, stdout, stderr))
	root.AddCommand(newDoctorCommand(&opts, stdout, stderr))
	root.AddCommand(newPlanCommand(&opts, stdout, stderr))
	root.AddCommand(newBuildCommand(&opts, stdout, stderr))
	root.AddCommand(newDiffCommand(&opts, stdout, stderr))
	root.AddCommand(newArtifactCommand(&opts, stdout, stderr))
	root.AddCommand(newExportCommand(&opts, stdout, stderr))
	return root
}

func newVersionCommand(opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print fbt version metadata",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return exitCode(runVersion(*opts, stdout, stderr))
		},
	}
}

func runVersion(opts options, stdout io.Writer, stderr io.Writer) int {
	if err := rejectSelect("version", opts); err != nil {
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
}

func newInitCommand(opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	var template string
	var force bool
	cmd := &cobra.Command{
		Use:   "init [PROJECT_NAME]",
		Short: "Create a new fbt project",
		Long: "Create a new fbt project with fs_project.yml and resource directories.\n\n" +
			"Templates such as support and incident include deterministic demo runners so\n" +
			"the project can be planned and built locally before replacing runner commands.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			callArgs := append([]string{}, args...)
			if template != "" {
				callArgs = append(callArgs, "--template", template)
			}
			if force {
				callArgs = append(callArgs, "--force")
			}
			return exitCode(runInit(*opts, callArgs, stdout, stderr))
		},
	}
	cmd.Flags().StringVar(&template, "template", "blank", "Project template")
	cmd.Flags().BoolVar(&force, "force", false, "Allow overwriting existing files")
	return cmd
}

func newDoctorCommand(opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check project and runner readiness",
		Long: "Check whether the current project can be used by fbt.\n\n" +
			"Doctor parses project config, checks local state access, resolves configured\n" +
			"runners, and initializes protocol-compatible runners when available.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return exitCode(runDoctor(*opts, args, stdout, stderr))
		},
	}
}

func newPlanCommand(opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Preview run, skip, and blocked transforms",
		Long: "Preview selected transform work without writing state or starting runners.\n\n" +
			"Plan compares project definitions, source fingerprints, previous state,\n" +
			"upstream artifact versions, and confidence requirements, then prints why\n" +
			"each selected transform will run, skip, or block.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return exitCode(runPlan(*opts, boolFlagArgs("force", force), stdout, stderr))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Preview a deliberate rebuild")
	return cmd
}

func newBuildCommand(opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build selected artifacts and write receipts",
		Long: "Build selected artifacts through external runners and write local receipts.\n\n" +
			"Build invokes protocol-compatible runners, validates output candidates,\n" +
			"commits immutable artifact versions, updates current artifact pointers,\n" +
			"and records run receipts under .fbt/state.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return exitCode(runBuild(*opts, boolFlagArgs("force", force), stdout, stderr))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Run selected clean transforms too")
	return cmd
}

func newDiffCommand(opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	var against string
	cmd := &cobra.Command{
		Use:   "diff TARGET",
		Short: "Compare current and previous artifact versions",
		Long:  "Compare a current artifact version with the previous version or an explicit artifact version reference.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			callArgs := append([]string{}, args...)
			if against != "" {
				callArgs = append(callArgs, "--against", against)
			}
			return exitCode(runDiff(*opts, callArgs, stdout, stderr))
		},
	}
	cmd.Flags().StringVar(&against, "against", "", "Version to compare against")
	return cmd
}

func newArtifactCommand(opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Inspect artifact paths, versions, and lineage",
		Long:  "Inspect generated artifacts, current versions, history, lineage reasoning, and local storage growth.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return exitCode(runArtifact(*opts, []string{"ls"}, stdout, stderr))
		},
	}
	cmd.AddCommand(newArtifactSubcommand("ls", "", "List artifacts with recorded versions", cobra.NoArgs, opts, stdout, stderr))
	cmd.AddCommand(newArtifactSubcommand("path", "TARGET", "Print current logical and immutable artifact paths", cobra.ExactArgs(1), opts, stdout, stderr))
	cmd.AddCommand(newArtifactSubcommand("show", "TARGET", "Show the current artifact version and metadata", cobra.ExactArgs(1), opts, stdout, stderr))
	cmd.AddCommand(newArtifactSubcommand("explain", "TARGET", "Explain why an artifact will run, skip, or block", cobra.ExactArgs(1), opts, stdout, stderr))
	cmd.AddCommand(newArtifactSubcommand("history", "TARGET", "List recorded versions for an artifact", cobra.ExactArgs(1), opts, stdout, stderr))
	cmd.AddCommand(newArtifactSubcommand("retention", "", "Report local state and artifact storage usage", cobra.NoArgs, opts, stdout, stderr))
	return cmd
}

func newArtifactSubcommand(name, useArgs, short string, args cobra.PositionalArgs, opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	use := name
	if useArgs != "" {
		use += " " + useArgs
	}
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  args,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			callArgs := append([]string{name}, cmdArgs...)
			return exitCode(runArtifact(*opts, callArgs, stdout, stderr))
		},
	}
}

func newExportCommand(opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Write standard lineage or trace records",
		Long: "Write standard records from local fbt state.\n\n" +
			"Use openlineage to produce RunEvent NDJSON for lineage tools, or otel to\n" +
			"produce OTLP/JSON traces for observability tools. Without --output, records\n" +
			"are written to stdout for normal shell piping.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newExportFormatCommand("openlineage", opts, stdout, stderr))
	cmd.AddCommand(newExportFormatCommand("otel", opts, stdout, stderr))
	return cmd
}

func newExportFormatCommand(format string, opts *options, stdout io.Writer, stderr io.Writer) *cobra.Command {
	var output string
	short := "Write " + format + " records from local run history"
	long := "Write " + format + " records from local fbt state.\n\n" +
		"Without --output, records are written to stdout. With --output, fbt writes\n" +
		"the file and prints a short summary for humans."
	if format == "openlineage" {
		short = "Write OpenLineage RunEvent NDJSON"
		long = "Write OpenLineage-compatible RunEvent NDJSON from local artifact lineage.\n\n" +
			"Without --output, events are written to stdout for piping into Marquez,\n" +
			"OpenMetadata ingestion, or another lineage backend."
	}
	if format == "otel" {
		short = "Write OTLP/JSON execution traces"
		long = "Write OpenTelemetry OTLP/JSON traces from local fbt run receipts.\n\n" +
			"Without --output, the trace payload is written to stdout for piping or\n" +
			"posting to an OTLP-compatible collector."
	}
	cmd := &cobra.Command{
		Use:   format,
		Short: short,
		Long:  long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			callArgs := []string{format}
			if output != "" {
				callArgs = append(callArgs, "--output", output)
			}
			return exitCode(runExport(*opts, callArgs, stdout, stderr))
		},
	}
	cmd.Flags().StringVar(&output, "output", "", "Output file path")
	return cmd
}

func boolFlagArgs(name string, value bool) []string {
	if !value {
		return nil
	}
	return []string{"--" + name}
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
		return printArtifactLookupError(ctx, diffOpts.Target, "diff", opts.JSON, stderr)
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
		fmt.Fprintln(stdout, "Export: openlineage")
		printDisplayRows(stdout, "  ", []displayRow{
			{Label: "Format", Value: "OpenLineage RunEvent NDJSON"},
			{Label: "Output", Value: exportOpts.OutputPath},
			{Label: "Events", Value: fmt.Sprintf("%d", len(events))},
			{Label: "Next", Value: "post or ingest this NDJSON with a Marquez/OpenMetadata-compatible workflow"},
		})
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
		fmt.Fprintln(stdout, "Export: otel")
		printDisplayRows(stdout, "  ", []displayRow{
			{Label: "Format", Value: "OpenTelemetry OTLP/JSON traces"},
			{Label: "Output", Value: exportOpts.OutputPath},
			{Label: "Spans", Value: fmt.Sprintf("%d", spanCount)},
			{Label: "Next", Value: "post this JSON to an OTLP-compatible collector or store it as evidence"},
		})
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
	buildOpts, err := parseBuildArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	result, err := buildmgr.RunBuild(context.Background(), buildmgr.Options{
		ProjectDir: opts.ProjectDir,
		StateDir:   opts.StateDir,
		Select:     opts.Select,
		Force:      buildOpts.Force,
		FBTVersion: versioninfo.Version,
	})
	if err != nil {
		printError("build", err, stderr, opts.JSON)
		if errors.Is(err, runnermgr.ErrCapabilityIncompatible) || errors.Is(err, runnermgr.ErrLockIncompatible) {
			return 6
		}
		if isSelectionError(err) {
			return 2
		}
		return 1
	}
	result.Plan = contextualizePlan(result.Plan, opts)
	if opts.JSON {
		writeJSON(stdout, map[string]any{"command": "build", "status": "success", "summary": result.Plan.Summary, "runs": result.Runs})
		if result.Plan.Summary.Blocked > 0 {
			return 3
		}
		return 0
	}
	printSummary(stdout, "Build", result.Plan.Summary)
	runNames := planNodeNames(result.Plan.Nodes)
	for _, run := range result.Runs {
		fmt.Fprintf(stdout, "%-7s %s\n", strings.ToUpper(run.Status), displayTransformName(run.TransformID, runNames))
		var rows []displayRow
		if run.TransformRunID != "" {
			rows = append(rows, displayRow{Label: "run", Value: shortenResourceID(run.TransformRunID)})
		}
		for _, committed := range run.CommittedArtifacts {
			target := shortenResourceID(committed.ArtifactID)
			if committed.LogicalPath != "" {
				rows = append(rows, displayRow{Label: "output", Value: target + " -> " + committed.LogicalPath})
			}
			if committed.VersionID != "" {
				rows = append(rows, displayRow{Label: "committed", Value: shortVersionID(committed.VersionID)})
			}
			rows = append(rows, displayRow{Label: "next", Value: contextualizeNextStep("fbt artifact show "+target, opts)})
		}
		if len(run.CommittedArtifacts) == 0 {
			for _, version := range run.CommittedVersions {
				rows = append(rows, displayRow{Label: "committed", Value: shortVersionID(version)})
			}
		}
		for _, result := range run.EvaluationDetails {
			rows = append(rows, displayRow{Label: "eval", Value: formatEvaluationResult(result)})
		}
		printDisplayRows(stdout, "        ", rows)
		fmt.Fprintln(stdout)
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

type buildOptions struct {
	Force bool
}

func parseBuildArgs(args []string) (buildOptions, error) {
	opts := buildOptions{}
	for _, arg := range args {
		switch arg {
		case "--force":
			opts.Force = true
		default:
			if strings.HasPrefix(arg, "-") {
				return buildOptions{}, fmt.Errorf("unknown build flag: %s", arg)
			}
			return buildOptions{}, fmt.Errorf("build accepts no arguments")
		}
	}
	return opts, nil
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
	printDoctorHuman(stdout, ctx.ParseResult.ProjectDir, ctx.Store.Dir, status, checks)
	return code
}

func printDoctorHuman(stdout io.Writer, projectDir, stateDir, status string, checks []doctorCheck) {
	fmt.Fprintf(stdout, "Doctor: %s\n", status)
	fmt.Fprintf(stdout, "Project: %s\n", projectDir)
	fmt.Fprintf(stdout, "State: %s\n", stateDir)
	fmt.Fprintln(stdout)
	printDoctorSection(stdout, "Project", doctorChecksForGroup(checks, "project"))
	printDoctorSection(stdout, "State", doctorChecksForGroup(checks, "state"))
	printDoctorRunners(stdout, checks)
	if other := doctorChecksForGroup(checks, "other"); len(other) > 0 {
		printDoctorSection(stdout, "Other", other)
	}
}

func printDoctorSection(stdout io.Writer, title string, checks []doctorCheck) {
	if len(checks) == 0 {
		return
	}
	fmt.Fprintln(stdout, title)
	for _, check := range checks {
		fmt.Fprintf(stdout, "  %s %s: %s\n", check.Status, check.Code, check.Message)
	}
	fmt.Fprintln(stdout)
}

type doctorRunnerGroup struct {
	Name   string
	Checks []doctorCheck
}

func printDoctorRunners(stdout io.Writer, checks []doctorCheck) {
	groups := doctorRunnerGroups(checks)
	if len(groups) == 0 {
		return
	}
	fmt.Fprintln(stdout, "Runners")
	for _, group := range groups {
		fmt.Fprintf(stdout, "  %s\n", group.Name)
		for _, check := range group.Checks {
			fmt.Fprintf(stdout, "    %s %s: %s\n", check.Status, check.Code, check.Message)
		}
	}
	fmt.Fprintln(stdout)
}

func doctorChecksForGroup(checks []doctorCheck, group string) []doctorCheck {
	var selected []doctorCheck
	for _, check := range checks {
		if doctorCheckGroup(check) == group {
			selected = append(selected, check)
		}
	}
	return selected
}

func doctorRunnerGroups(checks []doctorCheck) []doctorRunnerGroup {
	indexByName := map[string]int{}
	var groups []doctorRunnerGroup
	for _, check := range checks {
		if doctorCheckGroup(check) != "runner" {
			continue
		}
		name := doctorRunnerName(check.Name)
		index, ok := indexByName[name]
		if !ok {
			index = len(groups)
			indexByName[name] = index
			groups = append(groups, doctorRunnerGroup{Name: name})
		}
		groups[index].Checks = append(groups[index].Checks, check)
	}
	return groups
}

func doctorCheckGroup(check doctorCheck) string {
	switch {
	case check.Name == "project_config":
		return "project"
	case strings.HasPrefix(check.Name, "state_"):
		return "state"
	case check.Name == "runner_discovery" || strings.HasPrefix(check.Name, "runner."):
		return "runner"
	default:
		return "other"
	}
}

func doctorRunnerName(checkName string) string {
	if checkName == "runner_discovery" {
		return "discovery"
	}
	name := strings.TrimPrefix(checkName, "runner.")
	if name == "" || name == checkName {
		return "unknown"
	}
	return name
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
	lock, lockPresent, lockErr := runnermgr.ReadLockfile(projectDir)
	if lockErr != nil {
		return append(runnerDoctorChecksWithoutLock(resolved, m), doctorError("runner_lock", "RUNNER_LOCK_SCHEMA_UNSUPPORTED", lockErr.Error()))
	}
	if len(resolved) == 0 {
		checks := []doctorCheck{doctorOK("runner_discovery", "RUNNER_DISCOVERY_EMPTY", "no runners configured")}
		if lockPresent {
			checks = append(checks, runnerLockCoverageChecks(lock, resolved)...)
		}
		return checks
	}
	var checks []doctorCheck
	if lockPresent {
		checks = append(checks, runnerLockCoverageChecks(lock, resolved)...)
	}
	for _, runner := range resolved {
		diagnostics := runnermgr.Diagnose(runner)
		lockDiagnostics := []runnermgr.Diagnostic{}
		if lockPresent {
			lockDiagnostics = append(lockDiagnostics, runnermgr.ValidateLockResolved(lock, runner)...)
			checks = append(checks, runnerDiagnosticsAsDoctorChecks(runner.Name, lockDiagnostics)...)
		}
		if runnermgr.HasErrors(diagnostics) {
			checks = append(checks, runnerDiagnosticsAsDoctorChecks(runner.Name, diagnostics)...)
			continue
		}
		checks = append(checks, runnerDiagnosticsAsDoctorChecks(runner.Name, diagnostics)...)
		protocolChecks, protocolDiagnostics := runnerProtocolDoctorChecks(runner, capabilityRequirementsForRunner(m, runner.Name), lock, lockPresent)
		checks = append(checks, protocolChecks...)
		if lockPresent {
			lockDiagnostics = append(lockDiagnostics, protocolDiagnostics...)
			if _, ok := lock.Runners[runner.Name]; ok && !runnermgr.HasErrors(lockDiagnostics) {
				checks = append(checks, runnerDiagnosticAsDoctorCheck(runner.Name, runnermgr.LockOKDiagnostic()))
			}
		}
	}
	return checks
}

func runnerDoctorChecksWithoutLock(resolved []runnermgr.Resolved, m manifest.Manifest) []doctorCheck {
	if len(resolved) == 0 {
		return []doctorCheck{doctorOK("runner_discovery", "RUNNER_DISCOVERY_EMPTY", "no runners configured")}
	}
	var checks []doctorCheck
	for _, runner := range resolved {
		diagnostics := runnermgr.Diagnose(runner)
		checks = append(checks, runnerDiagnosticsAsDoctorChecks(runner.Name, diagnostics)...)
		if runnermgr.HasErrors(diagnostics) {
			continue
		}
		protocolChecks, _ := runnerProtocolDoctorChecks(runner, capabilityRequirementsForRunner(m, runner.Name), runnermgr.Lockfile{}, false)
		checks = append(checks, protocolChecks...)
	}
	return checks
}

func runnerLockCoverageChecks(lock runnermgr.Lockfile, resolved []runnermgr.Resolved) []doctorCheck {
	var checks []doctorCheck
	for _, diagnostic := range runnermgr.ValidateLockCoverage(lock, resolved) {
		checks = append(checks, runnerDiagnosticAsDoctorCheck(diagnostic.RunnerName, diagnostic.Diagnostic))
	}
	return checks
}

func runnerDiagnosticsAsDoctorChecks(runnerName string, diagnostics []runnermgr.Diagnostic) []doctorCheck {
	var checks []doctorCheck
	for _, diagnostic := range diagnostics {
		checks = append(checks, runnerDiagnosticAsDoctorCheck(runnerName, diagnostic))
	}
	return checks
}

func runnerDiagnosticAsDoctorCheck(runnerName string, diagnostic runnermgr.Diagnostic) doctorCheck {
	return doctorCheck{Name: "runner." + runnerName, Status: doctorStatus(diagnostic.Severity), Code: diagnostic.Code, Severity: diagnostic.Severity, Message: diagnostic.Message}
}

func doctorStatus(severity string) string {
	switch severity {
	case "error":
		return "error"
	case "warning":
		return "warning"
	default:
		return "ok"
	}
}

func runnerProtocolDoctorChecks(resolved runnermgr.Resolved, requirements []runnermgr.CapabilityRequirement, lock runnermgr.Lockfile, lockPresent bool) ([]doctorCheck, []runnermgr.Diagnostic) {
	var checks []doctorCheck
	initResult, diagnostics, ok := runnerProtocolInitializeResult(resolved)
	for _, diagnostic := range diagnostics {
		checks = append(checks, runnerDiagnosticAsDoctorCheck(resolved.Name, diagnostic))
	}
	if !ok {
		return checks, nil
	}
	capabilityDiagnostics := runnermgr.ValidateCapabilities(initResult, requirements)
	for _, diagnostic := range capabilityDiagnostics {
		checks = append(checks, runnerDiagnosticAsDoctorCheck(resolved.Name, diagnostic))
	}
	var lockDiagnostics []runnermgr.Diagnostic
	if lockPresent {
		lockDiagnostics = runnermgr.ValidateLockInitialized(lock, resolved.Name, initResult)
		checks = append(checks, runnerDiagnosticsAsDoctorChecks(resolved.Name, lockDiagnostics)...)
	}
	return checks, lockDiagnostics
}

func runnerProtocolInitializeResult(resolved runnermgr.Resolved) (protocol.InitializeResult, []runnermgr.Diagnostic, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := runnermgr.StartProtocolClient(ctx, resolved)
	if err != nil {
		return protocol.InitializeResult{}, []runnermgr.Diagnostic{{Severity: "error", Code: "RUNNER_PROTOCOL_START_ERROR", Message: err.Error()}}, false
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
		return protocol.InitializeResult{}, []runnermgr.Diagnostic{{Severity: "error", Code: "RUNNER_PROTOCOL_INIT_ERROR", Message: err.Error()}}, false
	}
	return initResult, []runnermgr.Diagnostic{{Severity: "info", Code: "RUNNER_PROTOCOL_OK", Message: "runner protocol initialize succeeded"}}, true
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
	planOpts, err := parsePlanArgs(args)
	if err != nil {
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
	plan := planner.Build(planner.Inputs{Manifest: ctx.Manifest, PreviousManifest: previous, State: snapshot, Selected: selected, Force: planOpts.Force})
	plan = contextualizePlan(plan, opts)
	if opts.JSON {
		writeJSON(stdout, map[string]any{"command": "plan", "status": "success", "summary": plan.Summary, "nodes": plan.Nodes})
		return 0
	}
	printSummary(stdout, "Plan", plan.Summary)
	for _, node := range plan.Nodes {
		printPlanNode(stdout, node)
	}
	return 0
}

type planOptions struct {
	Force bool
}

func parsePlanArgs(args []string) (planOptions, error) {
	opts := planOptions{}
	for _, arg := range args {
		switch arg {
		case "--force":
			opts.Force = true
		default:
			if strings.HasPrefix(arg, "-") {
				return planOptions{}, fmt.Errorf("unknown plan flag: %s", arg)
			}
			return planOptions{}, fmt.Errorf("plan accepts no arguments")
		}
	}
	return opts, nil
}

func printPlanNode(stdout io.Writer, node planner.Node) {
	fmt.Fprintf(stdout, "%-7s %s\n", actionLabel(node.Action), node.Name)
	var rows []displayRow
	for _, reason := range node.DirtyReasons {
		rows = append(rows, displayRow{Label: "because", Value: humanizeResourceIDs(reason)})
	}
	for _, change := range node.SourceChanges {
		rows = appendSourceChangeRows(rows, change)
	}
	for _, reason := range node.BlockedReasons {
		rows = append(rows, displayRow{Label: "blocked", Value: humanizeResourceIDs(reason)})
	}
	if len(node.Outputs) > 0 {
		rows = append(rows, displayRow{Label: "output", Value: strings.Join(shortResourceIDs(node.Outputs), ", ")})
	}
	for _, step := range node.NextSteps {
		rows = append(rows, displayRow{Label: "next", Value: step})
	}
	printDisplayRows(stdout, "        ", rows)
	fmt.Fprintln(stdout)
}

const sourceChangePathLimit = 8

func appendSourceChangeRows(rows []displayRow, change planner.SourceChange) []displayRow {
	label := change.Name
	if label == "" {
		label = shortenResourceID(change.SourceID)
	}
	rows = append(rows, displayRow{Label: "source", Value: label + " " + sourceChangeSummary(change)})
	rows = append(rows, sourceChangePathRows("added", change.Added)...)
	rows = append(rows, sourceChangePathRows("changed", change.Changed)...)
	rows = append(rows, sourceChangePathRows("removed", change.Removed)...)
	return rows
}

func sourceChangeSummary(change planner.SourceChange) string {
	return fmt.Sprintf("(added %d, changed %d, removed %d)", len(change.Added), len(change.Changed), len(change.Removed))
}

func sourceChangePathRows(label string, paths []string) []displayRow {
	if len(paths) == 0 {
		return nil
	}
	limit := sourceChangePathLimit
	if len(paths) < limit {
		limit = len(paths)
	}
	rows := make([]displayRow, 0, limit+1)
	for _, path := range paths[:limit] {
		rows = append(rows, displayRow{Label: label, Value: path})
	}
	if more := len(paths) - limit; more > 0 {
		rows = append(rows, displayRow{Label: label, Value: fmt.Sprintf("... %d more", more)})
	}
	return rows
}

func contextualizePlan(plan planner.Plan, opts options) planner.Plan {
	for i := range plan.Nodes {
		plan.Nodes[i].NextSteps = contextualizeNextSteps(plan.Nodes[i].NextSteps, opts)
	}
	return plan
}

func contextualizeNextSteps(steps []string, opts options) []string {
	if opts.ProjectDir == "" && opts.StateDir == "" {
		return steps
	}
	contextualized := make([]string, 0, len(steps))
	for _, step := range steps {
		contextualized = append(contextualized, contextualizeNextStep(step, opts))
	}
	return contextualized
}

func contextualizeNextStep(step string, opts options) string {
	if !strings.HasPrefix(step, "fbt ") {
		return step
	}
	var suffix []string
	if opts.ProjectDir != "" && !strings.Contains(step, "--project-dir") {
		suffix = append(suffix, "--project-dir", shellArg(opts.ProjectDir))
	}
	if opts.StateDir != "" && !strings.Contains(step, "--state-dir") {
		suffix = append(suffix, "--state-dir", shellArg(opts.StateDir))
	}
	if len(suffix) == 0 {
		return step
	}
	return step + " " + strings.Join(suffix, " ")
}

func shellArg(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r == '/' || r == '.' || r == '_' || r == '-' || r == ':' || r == '=' || r == '+' || r == ',' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func printSummary(stdout io.Writer, title string, summary planner.Summary) {
	fmt.Fprintf(stdout, "%s\n", title)
	printDisplayRows(stdout, "  ", []displayRow{
		{Label: "selected", Value: fmt.Sprintf("%d", summary.Selected)},
		{Label: "run", Value: fmt.Sprintf("%d", summary.Run)},
		{Label: "skipped", Value: fmt.Sprintf("%d", summary.Skipped)},
		{Label: "blocked", Value: fmt.Sprintf("%d", summary.Blocked)},
	})
	fmt.Fprintln(stdout)
}

type displayRow struct {
	Label string
	Value string
}

func printDisplayRows(stdout io.Writer, indent string, rows []displayRow) {
	width := 0
	for _, row := range rows {
		if len(row.Label) > width {
			width = len(row.Label)
		}
	}
	for _, row := range rows {
		fmt.Fprintf(stdout, "%s%-*s  %s\n", indent, width, row.Label, row.Value)
	}
}

func formatEvaluationResult(result state.EvaluationResult) string {
	parts := []string{shortenResourceID(result.EvalID)}
	if result.Status != "" {
		parts = append(parts, result.Status)
	}
	summary := strings.Join(parts, " ")
	if result.Reason != "" {
		summary += " (" + result.Reason + ")"
	}
	if result.Hint != "" {
		summary += "; " + shortEvaluationHint(result.Hint)
	}
	return summary
}

func shortEvaluationHint(hint string) string {
	const delegated = "Use an external judge transform that produces a report artifact when this should be an active quality gate."
	if hint == delegated {
		return "use external judge transform for active gate"
	}
	return hint
}

func actionLabel(action planner.Action) string {
	switch action {
	case planner.ActionRun:
		return "RUN"
	case planner.ActionSkip:
		return "SKIP"
	case planner.ActionBlocked:
		return "BLOCK"
	default:
		return strings.ToUpper(string(action))
	}
}

func planNodeNames(nodes []planner.Node) map[string]string {
	names := make(map[string]string, len(nodes))
	for _, node := range nodes {
		names[node.TransformID] = node.Name
	}
	return names
}

func displayTransformName(transformID string, names map[string]string) string {
	if name := names[transformID]; name != "" {
		return name
	}
	return shortenResourceID(transformID)
}

func shortResourceIDs(ids []string) []string {
	shortened := make([]string, 0, len(ids))
	for _, id := range ids {
		shortened = append(shortened, shortenResourceID(id))
	}
	return shortened
}

func humanizeResourceIDs(text string) string {
	parts := strings.Fields(text)
	for i, part := range parts {
		core := strings.TrimRight(part, ".,;:")
		suffix := strings.TrimPrefix(part, core)
		parts[i] = shortenResourceID(core) + suffix
	}
	return strings.Join(parts, " ")
}

func shortenResourceID(id string) string {
	if strings.HasPrefix(id, "artifact_version.") {
		return shortVersionID(id)
	}
	parts := strings.Split(id, ".")
	if len(parts) >= 3 {
		switch parts[0] {
		case "source", "runner":
			return strings.Join(parts[2:], ".")
		case "artifact", "transform", "policy", "eval", "transform_asset":
			return parts[len(parts)-1]
		case "transform_run":
			return parts[len(parts)-1]
		}
	}
	return id
}

func shortVersionID(versionID string) string {
	parts := strings.Split(versionID, ".")
	if len(parts) >= 4 && parts[0] == "artifact_version" {
		hash := parts[len(parts)-1]
		artifactName := parts[len(parts)-2]
		if strings.HasPrefix(hash, "sha256_") {
			hash = "sha256:" + strings.TrimPrefix(hash, "sha256_")
		}
		return artifactName + "@" + shortDigest(hash)
	}
	return shortenLongToken(versionID)
}

func shortDigest(digest string) string {
	switch {
	case strings.HasPrefix(digest, "sha256:"):
		return "sha256:" + shortenLongToken(strings.TrimPrefix(digest, "sha256:"))
	case strings.HasPrefix(digest, "sha256_"):
		return "sha256:" + shortenLongToken(strings.TrimPrefix(digest, "sha256_"))
	default:
		return shortenLongToken(digest)
	}
}

func shortenLongToken(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

func formatModel(model map[string]any) string {
	provider, _ := model["provider"].(string)
	name, _ := model["name"].(string)
	switch {
	case provider != "" && name != "":
		return provider + "/" + name
	case name != "":
		return name
	default:
		return compactJSON(model)
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
		fmt.Fprintln(stdout, "Artifacts")
		if len(ids) == 0 {
			fmt.Fprintln(stdout, "  none")
			return 0
		}
		for _, id := range ids {
			fmt.Fprintf(stdout, "  %s\n", shortenResourceID(id))
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
		return runArtifactExplain(ctx, args[1], opts, stdout, stderr)
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
	case "retention":
		if err := expectAtMostArgs("artifact retention", args, 1); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		return runArtifactRetention(ctx, opts.JSON, stdout, stderr)
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
		return printArtifactLookupError(ctx, target, "artifact path", jsonOutput, stderr)
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
		return printArtifactLookupError(ctx, target, "artifact show", jsonOutput, stderr)
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
		return printArtifactLookupError(ctx, target, "artifact history", jsonOutput, stderr)
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

func runArtifactRetention(ctx projectContext, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	report, err := state.BuildRetentionReport(ctx.ParseResult.ProjectDir, ctx.Store)
	if err != nil {
		printError("artifact retention", err, stderr, jsonOutput)
		return 5
	}
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": "artifact retention", "status": "success", "retention": report})
		return 0
	}
	printRetentionReport(stdout, ctx.ParseResult.ProjectDir, report)
	return 0
}

type artifactExplanation struct {
	ArtifactID     string                     `json:"artifact_id"`
	TransformID    string                     `json:"transform_id"`
	TransformName  string                     `json:"transform_name"`
	Action         planner.Action             `json:"action"`
	Decision       string                     `json:"decision"`
	Inputs         []manifest.TransformInput  `json:"inputs,omitempty"`
	Outputs        []manifest.TransformOutput `json:"outputs,omitempty"`
	Dependencies   []explanationDependency    `json:"dependencies,omitempty"`
	DirtyReasons   []string                   `json:"dirty_reasons,omitempty"`
	SourceChanges  []planner.SourceChange     `json:"source_changes,omitempty"`
	BlockedReasons []string                   `json:"blocked_reasons,omitempty"`
	NextSteps      []string                   `json:"next_steps,omitempty"`
	Current        *state.ArtifactPointer     `json:"current,omitempty"`
	PreviousRun    *state.LatestRun           `json:"previous_run,omitempty"`
}

type explanationDependency struct {
	Role                string `json:"role"`
	ResourceID          string `json:"resource_id"`
	Name                string `json:"name,omitempty"`
	Path                string `json:"path,omitempty"`
	Fingerprint         string `json:"fingerprint,omitempty"`
	PreviousFingerprint string `json:"previous_fingerprint,omitempty"`
	Changed             *bool  `json:"changed,omitempty"`
	CurrentVersionID    string `json:"current_version_id,omitempty"`
	Confidence          string `json:"confidence,omitempty"`
	RequiredConfidence  string `json:"required_confidence,omitempty"`
	EvalType            string `json:"eval_type,omitempty"`
	EvalStatus          string `json:"eval_status,omitempty"`
	EvalHint            string `json:"eval_hint,omitempty"`
}

func runArtifactExplain(ctx projectContext, target string, opts options, stdout io.Writer, stderr io.Writer) int {
	jsonOutput := opts.JSON
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
	evalResults, err := ctx.Store.ReadEvaluationResults()
	if err != nil {
		printError("artifact explain", err, stderr, jsonOutput)
		return 5
	}
	var currentPointer *state.ArtifactPointer
	var currentVersionID string
	if pointer, ok := snapshot.CurrentArtifacts[artifactID]; ok {
		copyPointer := pointer
		currentPointer = &copyPointer
		currentVersionID = pointer.CurrentVersionID
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
		Decision:       explanationDecision(node),
		Inputs:         transform.Inputs,
		Outputs:        transform.Outputs,
		Dependencies:   explanationDependencies(ctx.Manifest, previous, snapshot, transform, currentVersionID, evalResults),
		DirtyReasons:   node.DirtyReasons,
		SourceChanges:  node.SourceChanges,
		BlockedReasons: node.BlockedReasons,
		NextSteps:      contextualizeNextSteps(node.NextSteps, opts),
		Current:        currentPointer,
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

func explanationDecision(node planner.Node) string {
	switch node.Action {
	case planner.ActionRun:
		if len(node.DirtyReasons) == 0 {
			return "will build selected artifact"
		}
		return "will build because " + strings.Join(node.DirtyReasons, "; ")
	case planner.ActionBlocked:
		if len(node.BlockedReasons) == 0 {
			return "blocked"
		}
		return "blocked because " + strings.Join(node.BlockedReasons, "; ")
	case planner.ActionSkip:
		return "up to date; inspect the current artifact version"
	default:
		return string(node.Action)
	}
}

func explanationDependencies(current manifest.Manifest, previous *manifest.Manifest, snapshot state.Snapshot, transform manifest.TransformResource, currentVersionID string, evalResults state.EvaluationResultsIndex) []explanationDependency {
	var deps []explanationDependency
	for _, input := range transform.Inputs {
		dep := explanationDependency{Role: "input", ResourceID: input.UniqueID, Name: input.Name}
		if input.Kind == "source" {
			dep = sourceExplanationDependency(dep, current, previous, input.UniqueID)
		}
		if input.Kind == "ref" {
			dep = artifactRefExplanationDependency(dep, snapshot, input)
		}
		deps = append(deps, dep)
	}
	for _, assetID := range transform.Assets {
		deps = append(deps, assetExplanationDependency(current, previous, assetID))
	}
	if transform.Policy != "" {
		deps = append(deps, policyExplanationDependency(current, previous, transform.Policy))
	}
	for _, evalID := range transform.Evals {
		deps = append(deps, evalExplanationDependency(current, previous, evalID, currentVersionID, evalResults))
	}
	if transform.Runner != "" {
		deps = append(deps, runnerExplanationDependency(current, previous, transform.Runner))
	}
	return deps
}

func sourceExplanationDependency(dep explanationDependency, current manifest.Manifest, previous *manifest.Manifest, id string) explanationDependency {
	source, ok := current.Sources[id]
	if !ok {
		return dep
	}
	dep.Name = source.Name
	dep.Path = source.Path
	dep.Fingerprint = source.Fingerprint.Value
	if previous != nil {
		if prev, ok := previous.Sources[id]; ok {
			dep.PreviousFingerprint = prev.Fingerprint.Value
			dep.Changed = boolPtr(prev.Fingerprint.Value != source.Fingerprint.Value)
		} else {
			dep.Changed = boolPtr(true)
		}
	}
	return dep
}

func artifactRefExplanationDependency(dep explanationDependency, snapshot state.Snapshot, input manifest.TransformInput) explanationDependency {
	if required, ok := stringFromAnyMap(input.Require, "confidence"); ok {
		dep.RequiredConfidence = required
	}
	if pointer, ok := snapshot.CurrentArtifacts[input.UniqueID]; ok {
		dep.CurrentVersionID = pointer.CurrentVersionID
		dep.Confidence = pointer.Confidence
	}
	return dep
}

func assetExplanationDependency(current manifest.Manifest, previous *manifest.Manifest, id string) explanationDependency {
	dep := explanationDependency{Role: "asset", ResourceID: id}
	asset, ok := current.TransformAssets[id]
	if !ok {
		return dep
	}
	dep.Name = asset.Name
	dep.Path = asset.Path
	dep.Fingerprint = asset.Fingerprint.Value
	if previous != nil {
		if prev, ok := previous.TransformAssets[id]; ok {
			dep.PreviousFingerprint = prev.Fingerprint.Value
			dep.Changed = boolPtr(prev.Fingerprint.Value != asset.Fingerprint.Value)
		} else {
			dep.Changed = boolPtr(true)
		}
	}
	return dep
}

func policyExplanationDependency(current manifest.Manifest, previous *manifest.Manifest, id string) explanationDependency {
	dep := explanationDependency{Role: "policy", ResourceID: id}
	policy, ok := current.Policies[id]
	if !ok {
		return dep
	}
	dep.Name = policy.Name
	dep.Fingerprint = policy.Fingerprint.Value
	if previous != nil {
		if prev, ok := previous.Policies[id]; ok {
			dep.PreviousFingerprint = prev.Fingerprint.Value
			dep.Changed = boolPtr(prev.Fingerprint.Value != policy.Fingerprint.Value)
		} else {
			dep.Changed = boolPtr(true)
		}
	}
	return dep
}

func evalExplanationDependency(current manifest.Manifest, previous *manifest.Manifest, id string, currentVersionID string, evalResults state.EvaluationResultsIndex) explanationDependency {
	dep := explanationDependency{Role: "eval", ResourceID: id}
	eval, ok := current.Evals[id]
	if !ok {
		return dep
	}
	dep.Name = eval.Name
	dep.EvalType = eval.EvalType
	if eval.EvalType == "semantic" || eval.EvalType == "llm_judge" {
		dep.EvalStatus = "skipped"
		dep.EvalHint = "use external judge transform for active gate"
	}
	if result, ok := evaluationResultForVersion(evalResults, id, currentVersionID); ok {
		dep.EvalStatus = result.Status
		if result.Hint != "" {
			dep.EvalHint = shortEvaluationHint(result.Hint)
		}
	}
	dep.Fingerprint = eval.Fingerprint.Value
	if previous != nil {
		if prev, ok := previous.Evals[id]; ok {
			dep.PreviousFingerprint = prev.Fingerprint.Value
			dep.Changed = boolPtr(prev.Fingerprint.Value != eval.Fingerprint.Value)
		} else {
			dep.Changed = boolPtr(true)
		}
	}
	return dep
}

func evaluationResultForVersion(index state.EvaluationResultsIndex, evalID, artifactVersionID string) (state.EvaluationResult, bool) {
	if artifactVersionID == "" {
		return state.EvaluationResult{}, false
	}
	var best state.EvaluationResult
	for _, result := range index.EvaluationResults {
		if result.EvalID != evalID || result.ArtifactVersionID != artifactVersionID {
			continue
		}
		if best.ResultID == "" || result.ResultID > best.ResultID {
			best = result
		}
	}
	return best, best.ResultID != ""
}

func runnerExplanationDependency(current manifest.Manifest, previous *manifest.Manifest, id string) explanationDependency {
	dep := explanationDependency{Role: "runner", ResourceID: id}
	runner, ok := current.Runners[id]
	if !ok {
		return dep
	}
	dep.Name = runner.Name
	dep.Fingerprint = runner.Fingerprint.Value
	if previous != nil {
		if prev, ok := previous.Runners[id]; ok {
			dep.PreviousFingerprint = prev.Fingerprint.Value
			dep.Changed = boolPtr(prev.Fingerprint.Value != runner.Fingerprint.Value)
		} else {
			dep.Changed = boolPtr(true)
		}
	}
	return dep
}

func boolPtr(value bool) *bool {
	return &value
}

func stringFromAnyMap(values map[string]any, key string) (string, bool) {
	if values == nil {
		return "", false
	}
	value, ok := values[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}

func selectedTransformIDs(m manifest.Manifest, expr string) (map[string]struct{}, error) {
	return graph.SelectTransforms(m, expr)
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
	fmt.Fprintf(stdout, "Artifact path: %s\n", shortenResourceID(record.ArtifactID))
	rows := []displayRow{{Label: "Logical path", Value: record.LogicalPath}}
	if record.StoragePath != "" {
		rows = append(rows, displayRow{Label: "Immutable path", Value: record.StoragePath})
	}
	if record.VersionID != "" {
		rows = append(rows, displayRow{Label: "Version", Value: shortVersionID(record.VersionID)})
	} else {
		rows = append(rows, displayRow{Label: "Version", Value: "none"})
	}
	printDisplayRows(stdout, "  ", rows)
}

func printArtifactRecord(stdout io.Writer, record artifactRecord) {
	fmt.Fprintf(stdout, "Artifact: %s\n", shortenResourceID(record.ArtifactID))
	var artifactRows []displayRow
	if record.Current {
		artifactRows = append(artifactRows, displayRow{Label: "Status", Value: "current"})
	} else if record.VersionID != "" {
		artifactRows = append(artifactRows, displayRow{Label: "Status", Value: "historical"})
	}
	if record.LogicalPath != "" {
		artifactRows = append(artifactRows, displayRow{Label: "Path", Value: record.LogicalPath})
	}
	if record.VersionID != "" {
		artifactRows = append(artifactRows, displayRow{Label: "Version", Value: shortVersionID(record.VersionID)})
	}
	if record.Confidence != "" {
		artifactRows = append(artifactRows, displayRow{Label: "Confidence", Value: record.Confidence})
	}
	printDisplayRows(stdout, "  ", artifactRows)

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Build")
	var buildRows []displayRow
	if record.Runner != "" {
		buildRows = append(buildRows, displayRow{Label: "Runner", Value: shortenResourceID(record.Runner)})
	}
	if len(record.Model) > 0 {
		buildRows = append(buildRows, displayRow{Label: "Model", Value: formatModel(record.Model)})
	}
	if record.GeneratedBy != "" {
		buildRows = append(buildRows, displayRow{Label: "Run", Value: shortenResourceID(record.GeneratedBy)})
	}
	if record.CommittedAt != "" {
		buildRows = append(buildRows, displayRow{Label: "Committed", Value: record.CommittedAt})
	}
	printDisplayRows(stdout, "  ", buildRows)

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Details")
	detailRows := []displayRow{{Label: "Artifact ID", Value: record.ArtifactID}}
	if record.VersionID != "" {
		detailRows = append(detailRows, displayRow{Label: "Version ID", Value: record.VersionID})
	}
	if record.Digest != "" {
		detailRows = append(detailRows, displayRow{Label: "Digest", Value: record.Digest})
	}
	if record.ArtifactType != "" {
		detailRows = append(detailRows, displayRow{Label: "Type", Value: record.ArtifactType})
	}
	if record.StoragePath != "" {
		detailRows = append(detailRows, displayRow{Label: "Immutable path", Value: record.StoragePath})
	}
	if len(record.SemanticDescriptor) > 0 {
		detailRows = append(detailRows, displayRow{Label: "Semantic summary", Value: semanticDescriptorSummary(record.SemanticDescriptor)})
	}
	printDisplayRows(stdout, "  ", detailRows)
	for _, material := range record.Materials {
		value := humanizeResourceIDs(material.ResourceID)
		if material.ArtifactVersion != "" {
			value += " " + shortVersionID(material.ArtifactVersion)
		}
		if material.Digest != "" {
			value += " " + material.Digest
		}
		printDisplayRows(stdout, "  ", []displayRow{{Label: "Material", Value: value}})
	}
	fmt.Fprintln(stdout)
}

func printRetentionReport(stdout io.Writer, projectDir string, report state.RetentionReport) {
	fmt.Fprintln(stdout, "Artifact retention")
	rows := []displayRow{
		{Label: "Policy", Value: report.Policy},
		{Label: "State dir", Value: projectRelativeOrOriginal(projectDir, report.StateDir)},
		{Label: "Artifact dir", Value: projectRelativeOrOriginal(projectDir, report.ArtifactDir)},
		{Label: "State size", Value: humanBytes(report.StateBytes)},
		{Label: "Artifact size", Value: humanBytes(report.ArtifactBytes)},
		{Label: "Run records", Value: fmt.Sprintf("%d", report.RunRecords)},
		{Label: "Artifact versions", Value: fmt.Sprintf("%d", report.ArtifactVersions)},
		{Label: "Current versions", Value: fmt.Sprintf("%d", report.CurrentVersions)},
		{Label: "Historical versions", Value: fmt.Sprintf("%d", report.HistoricalVersions)},
		{Label: "Missing storage", Value: fmt.Sprintf("%d", len(report.MissingStorage))},
		{Label: "Action", Value: "no files removed; archive state and artifact dirs together"},
	}
	printDisplayRows(stdout, "  ", rows)
	for _, versionID := range report.MissingStorage {
		printDisplayRows(stdout, "  ", []displayRow{{Label: "Missing version", Value: versionID}})
	}
}

func projectRelativeOrOriginal(projectDir, path string) string {
	rel, err := filepath.Rel(projectDir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return path
	}
	return filepath.ToSlash(rel)
}

func compactJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func semanticDescriptorSummary(descriptor map[string]any) string {
	var parts []string
	if text, ok := mapValue(descriptor, "text_normalized_v1"); ok {
		values := []string{}
		if lines, ok := numberValue(text, "line_count"); ok {
			values = append(values, fmt.Sprintf("lines=%d", lines))
		}
		if words, ok := numberValue(text, "word_count"); ok {
			values = append(values, fmt.Sprintf("words=%d", words))
		}
		if chars, ok := numberValue(text, "char_count"); ok {
			values = append(values, fmt.Sprintf("chars=%d", chars))
		}
		if digest, ok := stringFromAnyMap(text, "digest"); ok {
			values = append(values, "digest="+shortDigest(digest))
		}
		if len(values) > 0 {
			parts = append(parts, "text "+strings.Join(values, " "))
		}
	}
	if markdown, ok := mapValue(descriptor, "markdown_ast_v1"); ok {
		values := []string{}
		if headings, ok := numberValue(markdown, "heading_count"); ok {
			values = append(values, fmt.Sprintf("headings=%d", headings))
		}
		if codeBlocks, ok := numberValue(markdown, "code_block_count"); ok {
			values = append(values, fmt.Sprintf("code_blocks=%d", codeBlocks))
		}
		if digest, ok := stringFromAnyMap(markdown, "digest"); ok {
			values = append(values, "digest="+shortDigest(digest))
		}
		if len(values) > 0 {
			parts = append(parts, "markdown "+strings.Join(values, " "))
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d descriptor(s)", len(descriptor))
	}
	return strings.Join(parts, "; ")
}

func mapValue(values map[string]any, key string) (map[string]any, bool) {
	value, ok := values[key]
	if !ok {
		return nil, false
	}
	mapped, ok := value.(map[string]any)
	return mapped, ok
}

func numberValue(values map[string]any, key string) (int64, bool) {
	value, ok := values[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	default:
		return 0, false
	}
}

func humanBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	for _, unit := range units {
		value = value / 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%.1f PiB", value/1024)
}

func printArtifactExplanation(stdout io.Writer, explanation artifactExplanation) {
	fmt.Fprintf(stdout, "Artifact: %s\n", shortenResourceID(explanation.ArtifactID))
	fmt.Fprintf(stdout, "Decision: %s\n", actionLabel(explanation.Action))
	if explanation.Decision != "" {
		printDisplayRows(stdout, "  ", []displayRow{{Label: "Reason", Value: humanizeResourceIDs(shortDecisionReason(explanation.Decision))}})
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Producer")
	rows := []displayRow{{Label: "Transform", Value: explanation.TransformName}}
	if explanation.Current != nil {
		rows = append(rows,
			displayRow{Label: "Current version", Value: shortVersionID(explanation.Current.CurrentVersionID)},
			displayRow{Label: "Confidence", Value: explanation.Current.Confidence},
		)
	} else {
		rows = append(rows, displayRow{Label: "Current version", Value: "none"})
	}
	if explanation.PreviousRun != nil {
		rows = append(rows, displayRow{Label: "Previous run", Value: shortenResourceID(explanation.PreviousRun.LatestRunID) + " " + explanation.PreviousRun.LatestStatus})
	} else {
		rows = append(rows, displayRow{Label: "Previous run", Value: "none"})
	}
	printDisplayRows(stdout, "  ", rows)
	if len(explanation.Dependencies) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Inputs")
		printExplanationDependencies(stdout, explanation.Dependencies)
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Outputs")
	printExplanationOutputs(stdout, explanation.Outputs)
	if len(explanation.DirtyReasons) > 0 || len(explanation.BlockedReasons) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Reasons")
	}
	var reasonRows []displayRow
	for _, reason := range explanation.DirtyReasons {
		reasonRows = append(reasonRows, displayRow{Label: "because", Value: humanizeResourceIDs(reason)})
	}
	for _, reason := range explanation.BlockedReasons {
		reasonRows = append(reasonRows, displayRow{Label: "blocked", Value: humanizeResourceIDs(reason)})
	}
	printDisplayRows(stdout, "  ", reasonRows)
	if len(explanation.SourceChanges) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Source Changes")
		var sourceRows []displayRow
		for _, change := range explanation.SourceChanges {
			sourceRows = appendSourceChangeRows(sourceRows, change)
		}
		printDisplayRows(stdout, "  ", sourceRows)
	}
	if len(explanation.NextSteps) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Next")
	}
	for _, step := range explanation.NextSteps {
		fmt.Fprintf(stdout, "  %s\n", step)
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Details")
	printDisplayRows(stdout, "  ", []displayRow{
		{Label: "Artifact ID", Value: explanation.ArtifactID},
		{Label: "Transform ID", Value: explanation.TransformID},
	})
}

func printExplanationDependencies(stdout io.Writer, deps []explanationDependency) {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  Status\tRole\tResource\tDetails")
	for _, dep := range deps {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", dependencyStatus(dep), dep.Role, shortenResourceID(dep.ResourceID), dependencyDetails(dep))
	}
	tw.Flush()
}

func dependencyStatus(dep explanationDependency) string {
	status := "ok"
	if dep.Changed != nil && *dep.Changed {
		status = "changed"
	}
	if dep.Role == "eval" && dep.EvalStatus == "skipped" {
		status = "skipped"
	}
	if dep.Role == "input" && strings.HasPrefix(dep.ResourceID, "artifact.") && dep.CurrentVersionID == "" {
		status = "missing"
	}
	return status
}

func dependencyDetails(dep explanationDependency) string {
	var details []string
	if dep.Name != "" && dep.Name != shortenResourceID(dep.ResourceID) {
		details = append(details, "name="+dep.Name)
	}
	if dep.Path != "" {
		details = append(details, "path="+dep.Path)
	}
	if dep.CurrentVersionID != "" {
		details = append(details, "version="+shortVersionID(dep.CurrentVersionID))
	}
	if dep.Confidence != "" {
		details = append(details, "confidence="+dep.Confidence)
	}
	if dep.RequiredConfidence != "" {
		details = append(details, "requires="+dep.RequiredConfidence)
	}
	if dep.EvalType != "" {
		details = append(details, "type="+dep.EvalType)
	}
	if dep.EvalStatus != "" {
		details = append(details, "status="+dep.EvalStatus)
	}
	if dep.EvalHint != "" {
		details = append(details, "hint="+dep.EvalHint)
	}
	if dep.Fingerprint != "" {
		details = append(details, "fingerprint="+shortDigest(dep.Fingerprint))
	}
	return strings.Join(details, "  ")
}

func printExplanationOutputs(stdout io.Writer, outputs []manifest.TransformOutput) {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  Artifact\tPath")
	for _, output := range outputs {
		fmt.Fprintf(tw, "  %s\t%s\n", shortenResourceID(output.UniqueID), output.DeclaredPath)
	}
	tw.Flush()
}

func shortDecisionReason(decision string) string {
	for _, prefix := range []string{
		"will build because ",
		"blocked because ",
	} {
		if strings.HasPrefix(decision, prefix) {
			return strings.TrimPrefix(decision, prefix)
		}
	}
	return decision
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

func printArtifactLookupError(ctx projectContext, target, command string, jsonOutput bool, stderr io.Writer) int {
	artifactID, declared := resolveArtifactID(ctx.Manifest, target)
	if jsonOutput {
		payload := map[string]any{
			"command": command,
			"status":  "error",
			"error":   "artifact not found: " + target,
		}
		if declared {
			payload["error"] = "artifact has no built version yet: " + target
			payload["artifact_id"] = artifactID
			payload["hint"] = "run fbt build --select " + buildTargetForArtifact(ctx.Manifest, artifactID)
		}
		writeJSON(stderr, payload)
		return 2
	}
	if declared {
		fmt.Fprintf(stderr, "Error: artifact has no built version yet: %s\n", target)
		fmt.Fprintf(stderr, "Hint: run `fbt build --select %s` to create it.\n", buildTargetForArtifact(ctx.Manifest, artifactID))
		return 2
	}
	fmt.Fprintf(stderr, "Error: artifact not found: %s\n", target)
	fmt.Fprintln(stderr, "Hint: run `fbt artifact ls` to list recorded artifacts, or `fbt plan` to inspect declared transforms.")
	return 2
}

func buildTargetForArtifact(m manifest.Manifest, artifactID string) string {
	if transform, ok := producerTransform(m, artifactID); ok && transform.Name != "" {
		return transform.Name
	}
	return shortenResourceID(artifactID)
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
	hint := errorHint(err)
	if jsonOutput {
		payload := map[string]any{"command": command, "status": "error", "error": err.Error()}
		if hint != "" {
			payload["hint"] = hint
		}
		writeJSON(stderr, payload)
		return
	}
	printCLIError(stderr, err.Error())
}

func printCLIError(stderr io.Writer, message string) {
	fmt.Fprintf(stderr, "Error: %s\n", message)
	if hint := errorMessageHint(message); hint != "" {
		fmt.Fprintf(stderr, "Hint: %s\n", hint)
	}
}

func errorHint(err error) string {
	if err == nil {
		return ""
	}
	return errorMessageHint(err.Error())
}

func errorMessageHint(message string) string {
	switch {
	case strings.Contains(message, "selector matched no transforms"):
		return "run `fbt plan` without --select to see available transforms, or check selectors in fs_project.yml."
	case strings.Contains(message, "unknown selector"):
		return "use a transform name, tag:, path:, resource_type:, selector:, or graph form such as +target."
	case strings.Contains(message, "unknown flag: --dry-run") || strings.Contains(message, "unknown plan flag: --dry-run") || strings.Contains(message, "unknown build flag: --dry-run"):
		return "use `fbt plan` to preview without writing state or starting runners."
	case strings.Contains(message, "runner closed stdout before response") || strings.Contains(message, "failed to read runner response") || strings.Contains(message, "failed to write runner request") || strings.Contains(message, "runner stderr:"):
		return "check the configured runner command with `fbt doctor`; inspect runner stderr and verify credentials or CLI setup."
	case strings.Contains(message, "runner lock incompatible"):
		return "run `fbt doctor` to see runner lockfile drift, then update the runner installation or fbt.lock.json outside fbt."
	default:
		return ""
	}
}

func writeJSON(w io.Writer, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintf(w, `{"status":"error","error":%q}`+"\n", err.Error())
		return
	}
	fmt.Fprintf(w, "%s\n", data)
}
