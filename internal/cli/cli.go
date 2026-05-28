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

	"github.com/nyuta01/fbt/internal/approval"
	"github.com/nyuta01/fbt/internal/artifact"
	buildmgr "github.com/nyuta01/fbt/internal/build"
	"github.com/nyuta01/fbt/internal/config"
	diffmgr "github.com/nyuta01/fbt/internal/diff"
	docsgen "github.com/nyuta01/fbt/internal/docs"
	evalmgr "github.com/nyuta01/fbt/internal/eval"
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

var implementedCommands = []string{
	"parse",
	"init",
	"plan",
	"build",
	"eval",
	"review",
	"diff",
	"docs",
	"state",
	"artifact",
	"runner",
	"doctor",
	"export",
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
	case "parse":
		return runParse(opts, stdout, stderr)
	case "plan":
		return runPlan(opts, stdout, stderr)
	case "build":
		return runBuild(opts, stdout, stderr)
	case "eval":
		return runEval(opts, commandArgs[1:], stdout, stderr)
	case "review":
		return runReview(opts, commandArgs[1:], stdout, stderr)
	case "diff":
		return runDiff(opts, commandArgs[1:], stdout, stderr)
	case "docs":
		return runDocs(opts, commandArgs[1:], stdout, stderr)
	case "state":
		return runState(opts, commandArgs[1:], stdout, stderr)
	case "artifact":
		return runArtifact(opts, commandArgs[1:], stdout, stderr)
	case "runner":
		return runRunner(opts, commandArgs[1:], stdout, stderr)
	case "doctor":
		return runDoctor(opts, stdout, stderr)
	case "export":
		return runExport(opts, commandArgs[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", commandArgs[0])
		printHelp(stderr)
		return 2
	}
}

func runDiff(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
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

func runDocs(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	docsOpts, err := parseDocsArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	if docsOpts.Subcommand != "generate" {
		fmt.Fprintf(stderr, "unknown docs command: %s\n", docsOpts.Subcommand)
		return 2
	}
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	if err := ctx.Store.WriteManifest(ctx.Manifest); err != nil {
		printError("docs generate", err, stderr, opts.JSON)
		return 5
	}
	result, err := docsgen.Generate(ctx.ParseResult.ProjectDir, ctx.Manifest, ctx.Store, docsgen.Options{OutputDir: docsOpts.OutputDir})
	if err != nil {
		printError("docs generate", err, stderr, opts.JSON)
		return 5
	}
	if opts.JSON {
		writeJSON(stdout, map[string]any{"command": "docs generate", "status": "success", "docs": result})
		return 0
	}
	fmt.Fprintf(stdout, "Docs written to %s\n", result.IndexPath)
	return 0
}

type docsOptions struct {
	Subcommand string
	OutputDir  string
}

func parseDocsArgs(args []string) (docsOptions, error) {
	opts := docsOptions{Subcommand: "generate"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--output":
			i++
			if i >= len(args) {
				return docsOptions{}, fmt.Errorf("--output requires a value")
			}
			opts.OutputDir = args[i]
		case strings.HasPrefix(arg, "--output="):
			opts.OutputDir = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "--"):
			return docsOptions{}, fmt.Errorf("unknown docs flag: %s", arg)
		default:
			if opts.Subcommand != "generate" {
				return docsOptions{}, fmt.Errorf("docs accepts one subcommand")
			}
			opts.Subcommand = arg
		}
	}
	return opts, nil
}

func runExport(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
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
	approvals, err := ctx.Store.ReadApprovals()
	if err != nil {
		printError("export openlineage", err, stderr, opts.JSON)
		return 5
	}
	events := lineage.OpenLineageEvents(lineage.OpenLineageInput{
		Manifest:          ctx.Manifest,
		Snapshot:          snapshot,
		ArtifactVersions:  versions,
		EvaluationResults: evaluations,
		Approvals:         approvals,
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

func runEval(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Error: eval requires a target")
		return 2
	}
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	snapshot, err := ctx.Store.ReadState()
	if err != nil {
		printError("eval", err, stderr, opts.JSON)
		return 5
	}
	versions, err := ctx.Store.ReadArtifactVersions()
	if err != nil {
		printError("eval", err, stderr, opts.JSON)
		return 5
	}
	version, ok := findVersion(snapshot, versions, args[0])
	if !ok {
		fmt.Fprintf(stderr, "Error: artifact not found: %s\n", args[0])
		return 2
	}
	transform, ok := producerTransform(ctx.Manifest, version.ArtifactID)
	if !ok {
		fmt.Fprintf(stderr, "Error: producer transform not found for %s\n", version.ArtifactID)
		return 2
	}
	evalIDs, err := selectedEvalIDs(ctx.Manifest, transform.Evals, opts.Select)
	if err != nil {
		printError("eval", err, stderr, opts.JSON)
		return 2
	}
	transform.Evals = evalIDs
	outcome, evalErr := evalmgr.RunForCandidate(ctx.ParseResult.ProjectDir, transform, ctx.Manifest.Evals, version.VersionID, version.GeneratedBy, filepath.Join(ctx.ParseResult.ProjectDir, version.StoragePath))
	for _, result := range outcome.Results {
		if err := ctx.Store.PutEvaluationResult(result); err != nil {
			printError("eval", err, stderr, opts.JSON)
			return 5
		}
	}
	status := "success"
	code := 0
	if evalErr != nil {
		status = "failed"
		code = 1
	}
	if opts.JSON {
		writeJSON(stdout, map[string]any{"command": "eval", "status": status, "artifact_version_id": version.VersionID, "results": outcome.Results})
		return code
	}
	for _, result := range outcome.Results {
		fmt.Fprintf(stdout, "%s %s\n", result.Status, result.EvalID)
		if result.Score != nil {
			fmt.Fprintf(stdout, "  score: %.2f\n", *result.Score)
		}
		if result.GrantsConfidence != "" {
			fmt.Fprintf(stdout, "  grants_confidence: %s\n", result.GrantsConfidence)
		}
	}
	if evalErr != nil {
		fmt.Fprintf(stderr, "Error: %v\n", evalErr)
	}
	return code
}

func runReview(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	subcommand := "status"
	if len(args) > 0 {
		subcommand = args[0]
	}
	switch subcommand {
	case "status":
		if len(args) < 2 {
			return printAllReviewStatus(ctx.Store, opts.JSON, stdout, stderr)
		}
		status, err := approval.GetStatus(ctx.Store, args[1], "")
		if err != nil {
			printError("review status", err, stderr, opts.JSON)
			return 2
		}
		printReviewStatus(status, opts.JSON, stdout)
		return 0
	case "show":
		target, versionID, _, err := parseReviewArgs(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		return runReviewShow(ctx, target, versionID, opts.JSON, stdout, stderr)
	case "approve", "reject":
		target, versionID, comment, err := parseReviewArgs(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		reviewer := os.Getenv("USER")
		if reviewer == "" {
			reviewer = "local"
		}
		var status approval.Status
		if subcommand == "approve" {
			status, err = approval.Approve(ctx.Store, target, versionID, reviewer, comment)
		} else {
			status, err = approval.Reject(ctx.Store, target, versionID, reviewer, comment)
		}
		if err != nil {
			printError("review "+subcommand, err, stderr, opts.JSON)
			return 2
		}
		printReviewStatus(status, opts.JSON, stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown review command: %s\n", subcommand)
		return 2
	}
}

type reviewDetails struct {
	Review        approval.Status `json:"review"`
	Artifact      artifactRecord  `json:"artifact"`
	DiffAvailable bool            `json:"diff_available"`
	Commands      []string        `json:"commands"`
}

func runReviewShow(ctx projectContext, target, versionID string, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	status, err := approval.GetStatus(ctx.Store, target, versionID)
	if err != nil {
		printError("review show", err, stderr, jsonOutput)
		return 2
	}
	versions, err := ctx.Store.ReadArtifactVersions()
	if err != nil {
		printError("review show", err, stderr, jsonOutput)
		return 5
	}
	snapshot, approvals, err := readArtifactState(ctx.Store)
	if err != nil {
		printError("review show", err, stderr, jsonOutput)
		return 5
	}
	version, ok := versions.ArtifactVersions[status.ArtifactVersionID]
	if !ok {
		fmt.Fprintf(stderr, "Error: artifact version not found: %s\n", status.ArtifactVersionID)
		return 2
	}
	record := buildArtifactRecord(ctx, snapshot, approvals, version)
	commandTarget := target
	if commandTarget == "" {
		commandTarget = status.ArtifactVersionID
	}
	_, diffAvailable := lastApprovedVersion(ctx.Store, versions, version)
	details := reviewDetails{
		Review:        status,
		Artifact:      record,
		DiffAvailable: diffAvailable,
		Commands:      reviewCommands(commandTarget, diffAvailable),
	}
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": "review show", "status": "success", "review": details})
		return 0
	}
	printReviewDetails(stdout, details)
	return 0
}

func reviewCommands(target string, diffAvailable bool) []string {
	commands := []string{
		fmt.Sprintf("fbt artifact show %s", target),
		fmt.Sprintf("fbt artifact path %s", target),
	}
	if diffAvailable {
		commands = append(commands, fmt.Sprintf("fbt diff %s --against last-approved", target))
	}
	commands = append(commands,
		fmt.Sprintf("fbt review approve %s --comment \"reviewed\"", target),
		fmt.Sprintf("fbt review reject %s --comment \"reason\"", target),
	)
	return commands
}

func printReviewDetails(stdout io.Writer, details reviewDetails) {
	fmt.Fprintf(stdout, "%s\n", details.Review.ArtifactID)
	fmt.Fprintf(stdout, "  version: %s\n", details.Review.ArtifactVersionID)
	fmt.Fprintf(stdout, "  status: %s\n", details.Review.Status)
	if details.Review.Confidence != "" {
		fmt.Fprintf(stdout, "  confidence: %s\n", details.Review.Confidence)
	}
	if details.Review.ReviewGroup != "" {
		fmt.Fprintf(stdout, "  group: %s\n", details.Review.ReviewGroup)
	}
	if details.Artifact.LogicalPath != "" {
		fmt.Fprintf(stdout, "  logical_path: %s\n", details.Artifact.LogicalPath)
	}
	if details.Artifact.StoragePath != "" {
		fmt.Fprintf(stdout, "  storage_path: %s\n", details.Artifact.StoragePath)
	}
	if details.Artifact.Digest != "" {
		fmt.Fprintf(stdout, "  digest: %s\n", details.Artifact.Digest)
	}
	if details.Artifact.Runner != "" {
		fmt.Fprintf(stdout, "  runner: %s\n", details.Artifact.Runner)
	}
	if len(details.Artifact.Model) > 0 {
		fmt.Fprintf(stdout, "  model: %s\n", compactJSON(details.Artifact.Model))
	}
	if details.Artifact.GeneratedBy != "" {
		fmt.Fprintf(stdout, "  generated_by: %s\n", details.Artifact.GeneratedBy)
	}
	for _, command := range details.Commands {
		fmt.Fprintf(stdout, "  %s: %s\n", reviewCommandLabel(command), command)
	}
}

func reviewCommandLabel(command string) string {
	switch {
	case strings.Contains(command, " review approve "):
		return "approve_after_review"
	case strings.Contains(command, " review reject "):
		return "reject_after_review"
	default:
		return "inspect"
	}
}

func runBuild(opts options, stdout io.Writer, stderr io.Writer) int {
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

func runRunner(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	discovery := runnermgr.NewDiscovery(ctx.ParseResult.ProjectDir, ctx.ParseResult.Config)
	subcommand := "list"
	if len(args) > 0 {
		subcommand = args[0]
	}
	switch subcommand {
	case "list":
		resolved, err := discovery.List()
		if err != nil {
			printError("runner list", err, stderr, opts.JSON)
			return 6
		}
		if opts.JSON {
			writeJSON(stdout, map[string]any{"command": "runner list", "status": "success", "runners": resolved})
			return 0
		}
		for _, runner := range resolved {
			fmt.Fprintf(stdout, "%s\n", runner.Name)
			fmt.Fprintf(stdout, "  source: %s\n", runner.Source)
			fmt.Fprintf(stdout, "  command: %s\n", runner.Command)
			if len(runner.Args) > 0 {
				fmt.Fprintf(stdout, "  args: %s\n", strings.Join(runner.Args, " "))
			}
			if runner.CWD != "" {
				fmt.Fprintf(stdout, "  cwd: %s\n", runner.CWD)
			}
			if len(runner.Env) > 0 {
				fmt.Fprintf(stdout, "  env: %s\n", strings.Join(runner.Env, ", "))
			}
			for _, diagnostic := range runner.Diagnostics {
				fmt.Fprintf(stdout, "  %s: %s\n", diagnostic.Code, diagnostic.Message)
			}
		}
		return 0
	case "doctor", "validate":
		if len(args) < 2 {
			fmt.Fprintf(stderr, "Error: runner %s requires a runner name\n", subcommand)
			return 2
		}
		resolved, err := discovery.Resolve(args[1])
		if err != nil {
			printError("runner "+subcommand, err, stderr, opts.JSON)
			return 6
		}
		diagnostics := runnermgr.Diagnose(resolved)
		if !runnermgr.HasErrors(diagnostics) {
			diagnostics = append(diagnostics, runnerProtocolDiagnostics(resolved, capabilityRequirementsForRunner(ctx.Manifest, resolved.Name))...)
		}
		status := "success"
		code := 0
		if runnermgr.HasErrors(diagnostics) {
			status = "error"
			code = 6
		}
		if opts.JSON {
			writeJSON(stdout, map[string]any{"command": "runner " + subcommand, "status": status, "runner": resolved, "diagnostics": diagnostics})
			return code
		}
		fmt.Fprintf(stdout, "%s\n", resolved.Name)
		fmt.Fprintf(stdout, "  source: %s\n", resolved.Source)
		fmt.Fprintf(stdout, "  command: %s\n", resolved.Command)
		if len(resolved.Args) > 0 {
			fmt.Fprintf(stdout, "  args: %s\n", strings.Join(resolved.Args, " "))
		}
		if resolved.CWD != "" {
			fmt.Fprintf(stdout, "  cwd: %s\n", resolved.CWD)
		}
		if len(resolved.Env) > 0 {
			fmt.Fprintf(stdout, "  env: %s\n", strings.Join(resolved.Env, ", "))
		}
		for _, diagnostic := range diagnostics {
			fmt.Fprintf(stdout, "  %s: %s\n", diagnostic.Code, diagnostic.Message)
		}
		return code
	default:
		fmt.Fprintf(stderr, "unknown runner command: %s\n", subcommand)
		return 2
	}
}

type doctorCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

func runDoctor(opts options, stdout io.Writer, stderr io.Writer) int {
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
				checks = append(checks, doctorCheck{Name: "runner." + runner.Name, Status: "error", Code: diagnostic.Code, Severity: diagnostic.Severity, Message: diagnostic.Message})
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
	fmt.Fprintln(w, "Implemented commands:")
	fmt.Fprintln(w, "  help       Show this help")
	fmt.Fprintln(w, "  version    Print version")
	for _, command := range implementedCommands {
		fmt.Fprintf(w, "  %-10s\n", command)
	}
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

func runParse(opts options, stdout io.Writer, stderr io.Writer) int {
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	if err := ctx.Store.WriteManifest(ctx.Manifest); err != nil {
		printError("parse", err, stderr, opts.JSON)
		return 5
	}
	if opts.JSON {
		writeJSON(stdout, map[string]any{
			"command": "parse",
			"status":  "success",
			"summary": map[string]any{
				"sources":          len(ctx.Manifest.Sources),
				"artifacts":        len(ctx.Manifest.Artifacts),
				"transforms":       len(ctx.Manifest.Transforms),
				"transform_assets": len(ctx.Manifest.TransformAssets),
				"policies":         len(ctx.Manifest.Policies),
				"evals":            len(ctx.Manifest.Evals),
				"runners":          len(ctx.Manifest.Runners),
			},
			"manifest_path": filepath.Join(ctx.Store.Dir, "manifest.json"),
		})
		return 0
	}
	fmt.Fprintf(stdout, "Parsed %d resources\n", resourceCount(ctx.Manifest))
	fmt.Fprintf(stdout, "Manifest written to %s\n", filepath.Join(ctx.Store.Dir, "manifest.json"))
	return 0
}

func runPlan(opts options, stdout io.Writer, stderr io.Writer) int {
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
	if err := ctx.Store.WriteManifest(ctx.Manifest); err != nil {
		printError("plan", err, stderr, opts.JSON)
		return 5
	}
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

func runState(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
	ctx, err := loadProject(opts)
	if err != nil {
		printParseError(err, ctx.ParseResult.Diagnostics, stderr, opts.JSON)
		return 2
	}
	subcommand := "status"
	if len(args) > 0 {
		subcommand = args[0]
	}
	switch subcommand {
	case "status":
		snapshot, err := ctx.Store.ReadState()
		if err != nil {
			printError("state", err, stderr, opts.JSON)
			return 5
		}
		versions, err := ctx.Store.ReadArtifactVersions()
		if err != nil {
			printError("state", err, stderr, opts.JSON)
			return 5
		}
		if opts.JSON {
			writeJSON(stdout, map[string]any{"command": "state status", "status": "success", "state_dir": ctx.Store.Dir, "current_artifacts": len(snapshot.CurrentArtifacts), "latest_runs": len(snapshot.LatestRuns), "artifact_versions": len(versions.ArtifactVersions)})
			return 0
		}
		fmt.Fprintf(stdout, "State dir: %s\n", ctx.Store.Dir)
		fmt.Fprintf(stdout, "Current artifacts: %d\n", len(snapshot.CurrentArtifacts))
		fmt.Fprintf(stdout, "Latest runs: %d\n", len(snapshot.LatestRuns))
		fmt.Fprintf(stdout, "Artifact versions: %d\n", len(versions.ArtifactVersions))
		return 0
	case "ls":
		files, err := stateFiles(ctx.Store.Dir)
		if err != nil {
			printError("state", err, stderr, opts.JSON)
			return 5
		}
		if opts.JSON {
			writeJSON(stdout, map[string]any{"command": "state ls", "status": "success", "files": files})
			return 0
		}
		for _, file := range files {
			fmt.Fprintln(stdout, file)
		}
		return 0
	case "current":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "Error: state current requires a target")
			return 2
		}
		return runStateCurrent(ctx.Store, args[1], opts.JSON, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown state command: %s\n", subcommand)
		return 2
	}
}

func runStateCurrent(store state.Store, target string, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	snapshot, err := store.ReadState()
	if err != nil {
		printError("state current", err, stderr, jsonOutput)
		return 5
	}
	id, pointer, ok := findPointer(snapshot, target)
	if !ok {
		fmt.Fprintf(stderr, "Error: current artifact not found: %s\n", target)
		return 2
	}
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": "state current", "status": "success", "artifact_id": id, "current": pointer})
		return 0
	}
	fmt.Fprintf(stdout, "%s\n", id)
	fmt.Fprintf(stdout, "  version: %s\n", pointer.CurrentVersionID)
	fmt.Fprintf(stdout, "  digest: %s\n", pointer.CurrentDigest)
	fmt.Fprintf(stdout, "  path: %s\n", pointer.LogicalPath)
	return 0
}

func runArtifact(opts options, args []string, stdout io.Writer, stderr io.Writer) int {
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
			fmt.Fprintln(stderr, "Error: artifact show requires a target")
			return 2
		}
		return runArtifactShow(ctx, versions, args[1], opts.JSON, stdout, stderr)
	case "explain":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "Error: artifact explain requires a target")
			return 2
		}
		return runArtifactExplain(ctx, args[1], opts.JSON, stdout, stderr)
	case "path":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "Error: artifact path requires a target")
			return 2
		}
		return runArtifactPath(ctx, versions, args[1], opts.JSON, stdout, stderr)
	case "history":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "Error: artifact history requires a target")
			return 2
		}
		return runArtifactHistory(ctx, versions, args[1], opts.JSON, stdout, stderr)
	case "versions":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "Error: artifact versions requires a target")
			return 2
		}
		return printArtifactVersion("artifact versions", versions, args[1], true, opts.JSON, stdout, stderr)
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
	ApprovalStatus      string              `json:"approval_status,omitempty"`
	Current             bool                `json:"current"`
	CreatedAt           string              `json:"created_at,omitempty"`
	CommittedAt         string              `json:"committed_at,omitempty"`
	Materials           []state.Material    `json:"materials,omitempty"`
	SemanticDescriptor  map[string]any      `json:"semantic_descriptor,omitempty"`
	Descriptor          artifact.Descriptor `json:"descriptor,omitempty"`
}

func runArtifactPath(ctx projectContext, versions state.ArtifactVersionsIndex, target string, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	snapshot, approvals, err := readArtifactState(ctx.Store)
	if err != nil {
		printError("artifact path", err, stderr, jsonOutput)
		return 5
	}
	if version, ok := findVersion(snapshot, versions, target); ok {
		record := buildArtifactRecord(ctx, snapshot, approvals, version)
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
	snapshot, approvals, err := readArtifactState(ctx.Store)
	if err != nil {
		printError("artifact show", err, stderr, jsonOutput)
		return 5
	}
	version, ok := findVersion(snapshot, versions, target)
	if !ok {
		fmt.Fprintf(stderr, "Error: artifact not found: %s\n", target)
		return 2
	}
	record := buildArtifactRecord(ctx, snapshot, approvals, version)
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": "artifact show", "status": "success", "artifact": record})
		return 0
	}
	printArtifactRecord(stdout, record)
	return 0
}

func runArtifactHistory(ctx projectContext, versions state.ArtifactVersionsIndex, target string, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	snapshot, approvals, err := readArtifactState(ctx.Store)
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
		records = append(records, buildArtifactRecord(ctx, snapshot, approvals, version))
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

func readArtifactState(store state.Store) (state.Snapshot, state.ApprovalIndex, error) {
	snapshot, err := store.ReadState()
	if err != nil {
		return state.Snapshot{}, state.ApprovalIndex{}, err
	}
	approvals, err := store.ReadApprovals()
	if err != nil {
		return state.Snapshot{}, state.ApprovalIndex{}, err
	}
	return snapshot, approvals, nil
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
	return selected, nil
}

func buildArtifactRecord(ctx projectContext, snapshot state.Snapshot, approvals state.ApprovalIndex, version state.ArtifactVersion) artifactRecord {
	record := artifactRecord{
		VersionID:           version.VersionID,
		ArtifactID:          version.ArtifactID,
		LogicalPath:         version.LogicalPath,
		StoragePath:         version.StoragePath,
		Digest:              version.Descriptor.Digest,
		ArtifactType:        version.Descriptor.ArtifactType,
		GeneratedBy:         version.GeneratedBy,
		Confidence:          version.Confidence,
		ApprovalStatus:      version.ApprovalStatus,
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
		if pointer.ApprovalStatus != "" {
			record.ApprovalStatus = pointer.ApprovalStatus
		}
	}
	if approvalRecord, ok := approvals.Approvals[version.VersionID]; ok && approvalRecord.Status != "" {
		record.ApprovalStatus = approvalRecord.Status
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
	if record.ApprovalStatus != "" {
		fmt.Fprintf(stdout, "  approval_status: %s\n", record.ApprovalStatus)
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
		fmt.Fprintf(stdout, "  approval_status: %s\n", explanation.Current.ApprovalStatus)
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

func printArtifactVersion(command string, index state.ArtifactVersionsIndex, target string, all bool, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	matches := matchingVersions(index, target)
	if len(matches) == 0 {
		fmt.Fprintf(stderr, "Error: artifact not found: %s\n", target)
		return 2
	}
	if !all && len(matches) > 1 {
		matches = matches[:1]
	}
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": command, "status": "success", "versions": matches})
		return 0
	}
	for _, version := range matches {
		fmt.Fprintf(stdout, "%s\n", version.VersionID)
		fmt.Fprintf(stdout, "  artifact: %s\n", version.ArtifactID)
		fmt.Fprintf(stdout, "  digest: %s\n", version.Descriptor.Digest)
		fmt.Fprintf(stdout, "  path: %s\n", version.LogicalPath)
	}
	return 0
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
	case "last-approved":
		version, ok := lastApprovedVersion(store, index, current)
		if !ok {
			return state.ArtifactVersion{}, fmt.Errorf("last approved artifact version not found for %s", current.ArtifactID)
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

func lastApprovedVersion(store state.Store, index state.ArtifactVersionsIndex, current state.ArtifactVersion) (state.ArtifactVersion, bool) {
	approvals, err := store.ReadApprovals()
	if err != nil {
		return state.ArtifactVersion{}, false
	}
	var approved []state.ArtifactVersion
	for versionID, approval := range approvals.Approvals {
		if approval.ArtifactID != current.ArtifactID || approval.Status != "approved" || versionID == current.VersionID {
			continue
		}
		version, ok := index.ArtifactVersions[versionID]
		if ok {
			approved = append(approved, version)
		}
	}
	sort.Slice(approved, func(i, j int) bool {
		return approved[i].VersionID < approved[j].VersionID
	})
	if len(approved) == 0 {
		return state.ArtifactVersion{}, false
	}
	return approved[len(approved)-1], true
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

func selectedEvalIDs(m manifest.Manifest, evalIDs []string, expr string) ([]string, error) {
	if expr == "" {
		return append([]string(nil), evalIDs...), nil
	}
	var selected []string
	for _, id := range evalIDs {
		evalResource := m.Evals[id]
		if id == expr || evalResource.Name == strings.Trim(expr, "+") {
			selected = append(selected, id)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("unknown eval selection: %s", expr)
	}
	return selected, nil
}

func parseReviewArgs(args []string) (target string, versionID string, comment string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--version":
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("--version requires a value")
			}
			versionID = args[i]
		case strings.HasPrefix(arg, "--version="):
			versionID = strings.TrimPrefix(arg, "--version=")
		case arg == "--comment":
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("--comment requires a value")
			}
			comment = args[i]
		case strings.HasPrefix(arg, "--comment="):
			comment = strings.TrimPrefix(arg, "--comment=")
		case strings.HasPrefix(arg, "--"):
			return "", "", "", fmt.Errorf("unknown review flag: %s", arg)
		default:
			if target != "" {
				return "", "", "", fmt.Errorf("review command accepts one target")
			}
			target = arg
		}
	}
	if target == "" && versionID == "" {
		return "", "", "", fmt.Errorf("review command requires a target")
	}
	return target, versionID, comment, nil
}

func printAllReviewStatus(store state.Store, jsonOutput bool, stdout io.Writer, stderr io.Writer) int {
	index, err := store.ReadApprovals()
	if err != nil {
		printError("review status", err, stderr, jsonOutput)
		return 5
	}
	ids := make([]string, 0, len(index.Approvals))
	for id := range index.Approvals {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if jsonOutput {
		approvals := make([]state.Approval, 0, len(ids))
		for _, id := range ids {
			approvals = append(approvals, index.Approvals[id])
		}
		writeJSON(stdout, map[string]any{"command": "review status", "status": "success", "approvals": approvals})
		return 0
	}
	for _, id := range ids {
		approval := index.Approvals[id]
		fmt.Fprintf(stdout, "%s\n", id)
		fmt.Fprintf(stdout, "  status: %s\n", approval.Status)
		if approval.ReviewGroup != "" {
			fmt.Fprintf(stdout, "  group: %s\n", approval.ReviewGroup)
		}
	}
	return 0
}

func printReviewStatus(status approval.Status, jsonOutput bool, stdout io.Writer) {
	if jsonOutput {
		writeJSON(stdout, map[string]any{"command": "review status", "status": "success", "review": status})
		return
	}
	fmt.Fprintf(stdout, "%s\n", status.ArtifactID)
	fmt.Fprintf(stdout, "  version: %s\n", status.ArtifactVersionID)
	fmt.Fprintf(stdout, "  status: %s\n", status.Status)
	if status.Confidence != "" {
		fmt.Fprintf(stdout, "  confidence: %s\n", status.Confidence)
	}
	if status.ReviewGroup != "" {
		fmt.Fprintf(stdout, "  group: %s\n", status.ReviewGroup)
	}
	if status.Status == "pending" {
		fmt.Fprintf(stdout, "  next: fbt review show %s\n", shortResourceName(status.ArtifactID))
	}
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

func stateFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	return files, nil
}

func findPointer(snapshot state.Snapshot, target string) (string, state.ArtifactPointer, bool) {
	for id, pointer := range snapshot.CurrentArtifacts {
		if id == target || strings.HasSuffix(id, "."+target) {
			return id, pointer, true
		}
	}
	return "", state.ArtifactPointer{}, false
}

func resourceCount(m manifest.Manifest) int {
	return len(m.Sources) + len(m.Artifacts) + len(m.Transforms) + len(m.TransformAssets) + len(m.Policies) + len(m.Evals) + len(m.Runners)
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
