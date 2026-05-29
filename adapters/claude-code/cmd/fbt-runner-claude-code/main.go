package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nyuta01/fbt/adapters/claude-code/internal/claudecodeadapter"
	"github.com/nyuta01/fbt/sdk/go/stdiojsonrpc"
)

func main() {
	if err := stdiojsonrpc.Serve(context.Background(), os.Stdin, os.Stdout, claudecodeadapter.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "fbt-runner-claude-code: %v\n", err)
		os.Exit(1)
	}
}
