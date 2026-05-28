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

	"github.com/nyuta01/fbt/internal/approval"
	buildmgr "github.com/nyuta01/fbt/internal/build"
	diffmgr "github.com/nyuta01/fbt/internal/diff"
	docsgen "github.com/nyuta01/fbt/internal/docs"
	evalmgr "github.com/nyuta01/fbt/internal/eval"
	"github.com/nyuta01/fbt/internal/graph"
	"github.com/nyuta01/fbt/internal/manifest"
	"github.com/nyuta01/fbt/internal/parser"
	"github.com/nyuta01/fbt/internal/planner"
	runnermgr "github.com/nyuta01/fbt/internal/runner"
	"github.com/nyuta01/fbt/internal/state"
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
}

var plannedCommands = []string{
	"run",
	"debug",
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
	default:
		if isPlannedCommand(commandArgs[0]) {
			fmt.Fprintf(stderr, "fbt %s: not implemented yet\n", commandArgs[0])
			return 2
		}
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

func runBuild(opts options, stdout io.Writer, stderr io.Writer) int {
	result, err := buildmgr.RunBuild(context.Background(), buildmgr.Options{
		ProjectDir: opts.ProjectDir,
		StateDir:   opts.StateDir,
		Select:     opts.Select,
		FBTVersion: versioninfo.Version,
	})
	if err != nil {
		printError("build", err, stderr, opts.JSON)
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
		for _, diagnostic := range diagnostics {
			fmt.Fprintf(stdout, "  %s: %s\n", diagnostic.Code, diagnostic.Message)
		}
		return code
	default:
		fmt.Fprintf(stderr, "unknown runner command: %s\n", subcommand)
		return 2
	}
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
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Planned commands:")
	for _, command := range plannedCommands {
		fmt.Fprintf(w, "  %s\n", command)
	}
}

func isPlannedCommand(command string) bool {
	for _, planned := range plannedCommands {
		if strings.EqualFold(command, planned) {
			return true
		}
	}
	return false
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
		return printArtifactVersion("artifact show", versions, args[1], false, opts.JSON, stdout, stderr)
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
		fmt.Fprintf(stderr, "  %s: %s\n", diagnostic.Code, diagnostic.Message)
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
