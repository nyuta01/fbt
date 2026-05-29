package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nyuta01/fbt/adapters/openai/internal/openaiadapter"
	"github.com/nyuta01/fbt/sdk/go/stdiojsonrpc"
)

func main() {
	if err := stdiojsonrpc.Serve(context.Background(), os.Stdin, os.Stdout, openaiadapter.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "fbt-runner-openai: %v\n", err)
		os.Exit(1)
	}
}
