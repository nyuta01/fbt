package redaction

import "strings"

const Placeholder = "[REDACTED]"

func String(value string, markers ...string) string {
	redacted := value
	for _, marker := range markers {
		if marker == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, marker, Placeholder)
	}
	return redacted
}

func Map(values map[string]any, keys ...string) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	deny := map[string]struct{}{}
	for _, key := range keys {
		deny[strings.ToLower(key)] = struct{}{}
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		if _, blocked := deny[strings.ToLower(key)]; blocked {
			out[key] = Placeholder
			continue
		}
		out[key] = value
	}
	return out
}
