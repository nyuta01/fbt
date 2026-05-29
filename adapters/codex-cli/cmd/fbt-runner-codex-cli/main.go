package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nyuta01/fbt/adapters/codex-cli/internal/codexcliadapter"
	"github.com/nyuta01/fbt/sdk/go/stdiojsonrpc"
)

func main() {
	if err := stdiojsonrpc.Serve(context.Background(), os.Stdin, os.Stdout, codexcliadapter.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "fbt-runner-codex-cli: %v\n", err)
		os.Exit(1)
	}
}
