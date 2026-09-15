package cli

import (
	"reflect"
	"testing"
)

var knownStringFlags = map[string]bool{"out": true}

func TestSplitArgsAndFlags(t *testing.T) {
	cases := []struct {
		name           string
		input          []string
		wantPositional []string
		wantFlags      []string
	}{
		{
			name:           "model only",
			input:          []string{"llama3"},
			wantPositional: []string{"llama3"},
			wantFlags:      nil,
		},
		{
			name:           "flag before model",
			input:          []string{"--out", "/tmp/blobs", "llama3"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"--out", "/tmp/blobs"},
		},
		{
			name:           "flag after model",
			input:          []string{"llama3", "--out", "/tmp/blobs"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"--out", "/tmp/blobs"},
		},
		{
			name:           "flag with inline = value",
			input:          []string{"llama3", "--out=/tmp/blobs"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"--out=/tmp/blobs"},
		},
		{
			name:           "boolean flag no value",
			input:          []string{"llama3", "--verbose"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"--verbose"},
		},
		{
			name:           "boolean flag before model",
			input:          []string{"--verbose", "llama3"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"--verbose"},
		},
		{
			name:           "multiple flags mixed with model",
			input:          []string{"--verbose", "llama3", "--out", "/tmp"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"--verbose", "--out", "/tmp"},
		},
		{
			name:           "single-dash flag",
			input:          []string{"llama3", "-v"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"-v"},
		},
		{
			name:           "single-dash flag with value",
			input:          []string{"-out", "/tmp", "llama3"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"-out", "/tmp"},
		},
		{
			name:           "no args",
			input:          []string{},
			wantPositional: nil,
			wantFlags:      nil,
		},
		{
			name:           "flags only no model",
			input:          []string{"--out", "/tmp"},
			wantPositional: nil,
			wantFlags:      []string{"--out", "/tmp"},
		},
		{
			name:           "two positional args",
			input:          []string{"llama3", "extra"},
			wantPositional: []string{"llama3", "extra"},
			wantFlags:      nil,
		},
		{
			name:           "flag at end with no value — not consumed",
			input:          []string{"llama3", "--out"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"--out"},
		},
		{
			name: "next token is a flag — not consumed as value",
			// --out is followed by --verbose, so --verbose stays as a flag,
			// not treated as the value of --out.
			input:          []string{"llama3", "--out", "--verbose"},
			wantPositional: []string{"llama3"},
			wantFlags:      []string{"--out", "--verbose"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotPositional, gotFlags := SplitArgsAndFlags(tc.input, knownStringFlags)
			if !reflect.DeepEqual(gotPositional, tc.wantPositional) {
				t.Errorf("positional:\n  got  %v\n  want %v", gotPositional, tc.wantPositional)
			}
			if !reflect.DeepEqual(gotFlags, tc.wantFlags) {
				t.Errorf("flags:\n  got  %v\n  want %v", gotFlags, tc.wantFlags)
			}
		})
	}
}
