package cli

import (
	"fmt"
	"io"
	"strings"
)

const Version = "0.0.0-dev"

var plannedCommands = []string{
	"init",
	"parse",
	"plan",
	"build",
	"run",
	"eval",
	"diff",
	"review",
	"docs",
	"state",
	"artifact",
	"runner",
	"debug",
}

// Run executes the current CLI surface. The MVP implementation is still
// intentionally tiny; this scaffold gives the harness a real executable target
// while product behavior is added behind spec-backed tasks.
func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "fbt %s\n", Version)
		return 0
	default:
		if isPlannedCommand(args[0]) {
			fmt.Fprintf(stderr, "fbt %s: not implemented yet\n", args[0])
			return 2
		}
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printHelp(stderr)
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
