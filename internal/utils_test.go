package internal

import (
	"testing"
)

func TestFormatParams_ValidJSON(t *testing.T) {
	in := `{"temperature":0.7,"num_ctx":4096,"stop":["<|eot_id|>"]}`
	want := "{\n  \"temperature\": 0.7,\n  \"num_ctx\": 4096,\n  \"stop\": [\n    \"<|eot_id|>\"\n  ]\n}"

	got := FormatParams(in)
	if got != want {
		t.Errorf("FormatParams(%q) = %q, want %q", in, got, want)
	}
}

func TestFormatParams_EmptyObject(t *testing.T) {
	got := FormatParams("{}")
	if got != "{}" {
		t.Errorf("FormatParams(%q) = %q, want %q", "{}", got, "{}")
	}
}

func TestFormatParams_InvalidJSONFallsBackToRaw(t *testing.T) {
	// Malformed/truncated blob: should be returned as-is (trimmed) rather
	// than dropped or causing a panic.
	in := "  not valid json  "
	want := "not valid json"
	got := FormatParams(in)
	if got != want {
		t.Errorf("FormatParams(%q) = %q, want %q", in, got, want)
	}
}
