package protocol

import (
	"encoding/json"
	"testing"
)

func TestInitializeResultShape(t *testing.T) {
	result := InitializeResult{
		Runner:   RunnerInfo{Name: "fbt-runner-example", Version: "0.1.0", Language: "go"},
		Protocol: ProtocolInfo{Version: Version, Framing: FramingJSONL},
		Capabilities: Capabilities{
			TransformTypes:   []string{"command"},
			ArtifactTypes:    []string{"markdown"},
			OutputCandidates: true,
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["runner"] == nil || decoded["protocol"] == nil || decoded["capabilities"] == nil {
		t.Fatalf("missing top-level fields: %s", data)
	}
}
