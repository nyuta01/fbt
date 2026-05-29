package stdiojsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nyuta01/fbt/sdk/go/protocol"
)

func TestServeWritesResponsesAndNotifications(t *testing.T) {
	input := strings.NewReader(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":"run","method":"fbt/runTransform","params":{}}` + "\n")
	var output bytes.Buffer

	err := Serve(context.Background(), input, &output, Handler{
		Initialize: func(context.Context, protocol.Request, *Writer) (any, error) {
			return protocol.InitializeResult{
				Runner:   protocol.RunnerInfo{Name: "test", Version: "0.1.0"},
				Protocol: protocol.ProtocolInfo{Version: protocol.Version, Framing: protocol.FramingJSONL},
				Capabilities: protocol.Capabilities{
					TransformTypes:   []string{"command"},
					ArtifactTypes:    []string{"markdown"},
					OutputCandidates: true,
				},
			}, nil
		},
		RunTransform: func(_ context.Context, req protocol.Request, writer *Writer) (any, error) {
			if err := writer.Notification(protocol.MethodEvent, protocol.Event{
				RequestID: req.ID, EventType: "progress", Message: "running",
			}); err != nil {
				return nil, err
			}
			return protocol.RunTransformResult{Status: "success", TransformRunID: "run_1"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 messages, got %d: %s", len(lines), output.String())
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &event); err != nil {
		t.Fatal(err)
	}
	if event["method"] != protocol.MethodEvent {
		t.Fatalf("expected event notification, got %+v", event)
	}
}
