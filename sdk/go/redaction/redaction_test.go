package redaction

import "testing"

func TestString(t *testing.T) {
	got := String("token=secret", "secret")
	if got != "token="+Placeholder {
		t.Fatalf("got %q", got)
	}
}

func TestMap(t *testing.T) {
	got := Map(map[string]any{"api_key": "secret", "model": "test"}, "api_key")
	if got["api_key"] != Placeholder || got["model"] != "test" {
		t.Fatalf("got %+v", got)
	}
}
