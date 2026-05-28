package runner

import (
	"context"
	"fmt"

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
	return protocol.Start(ctx, command, nil, protocol.Options{})
}
