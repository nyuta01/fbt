package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nyuta01/fbt/adapters/command/internal/commandadapter"
	"github.com/nyuta01/fbt/sdk/go/stdiojsonrpc"
)

func main() {
	if err := stdiojsonrpc.Serve(context.Background(), os.Stdin, os.Stdout, commandadapter.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "fbt-runner-command: %v\n", err)
		os.Exit(1)
	}
}
