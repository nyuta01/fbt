package runner

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/nyuta01/fbt/internal/protocol"
)

func StartProtocolClient(ctx context.Context, resolved Resolved) (*protocol.Client, error) {
	command := resolved.CommandPath
	if command == "" {
		command = resolved.Command
	}
	if command == "" {
		return nil, fmt.Errorf("runner command is empty")
	}
	return protocol.Start(ctx, command, resolved.Args, protocol.Options{
		Dir:            resolved.CWD,
		Env:            runnerEnv(resolved.Env),
		RedactEnvNames: resolved.Env,
	})
}

func runnerEnv(passThrough []string) []string {
	names := map[string]struct{}{}
	for _, name := range []string{"PATH", "HOME", "USER", "TMPDIR", "TMP", "TEMP", "SHELL"} {
		names[name] = struct{}{}
	}
	for _, name := range passThrough {
		names[name] = struct{}{}
	}
	sortedNames := make([]string, 0, len(names))
	for name := range names {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)
	env := make([]string, 0, len(names))
	for _, name := range sortedNames {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}
