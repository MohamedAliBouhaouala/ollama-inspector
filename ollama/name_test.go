package ollama

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseName_DefaultFilling(t *testing.T) {
	cases := []struct {
		input     string
		wantHost  string
		wantNS    string
		wantModel string
		wantTag   string
		wantFQ    bool
	}{
		{
			input: "llama3", wantHost: defaultHost, wantNS: defaultNamespace,
			wantModel: "llama3", wantTag: defaultTag, wantFQ: true,
		},
		{
			input: "mistral:7b", wantHost: defaultHost, wantNS: defaultNamespace,
			wantModel: "mistral", wantTag: "7b", wantFQ: true,
		},
		{
			input: "myns/mymodel:v2", wantHost: defaultHost, wantNS: "myns",
			wantModel: "mymodel", wantTag: "v2", wantFQ: true,
		},
		{
			input: "registry.ollama.ai/library/llama3:latest", wantHost: "registry.ollama.ai",
			wantNS: "library", wantModel: "llama3", wantTag: "latest", wantFQ: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			n := ParseName(tc.input)
			if n.Host != tc.wantHost {
				t.Errorf("Host: got %q, want %q", n.Host, tc.wantHost)
			}
			if n.Namespace != tc.wantNS {
				t.Errorf("Namespace: got %q, want %q", n.Namespace, tc.wantNS)
			}
			if n.Model != tc.wantModel {
				t.Errorf("Model: got %q, want %q", n.Model, tc.wantModel)
			}
			if n.Tag != tc.wantTag {
				t.Errorf("Tag: got %q, want %q", n.Tag, tc.wantTag)
			}
			if n.IsFullyQualified() != tc.wantFQ {
				t.Errorf("IsFullyQualified: got %v, want %v", n.IsFullyQualified(), tc.wantFQ)
			}
		})
	}
}

func TestDisplayShortest(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"llama3", "llama3:latest"},
		{"mistral:7b", "mistral:7b"},
		{"myns/mymodel:v1", "myns/mymodel:v1"},
		{"registry.ollama.ai/library/llama3:latest", "llama3:latest"},
		{"custom.host/mynamespace/mymodel:v1", "custom.host/mynamespace/mymodel:v1"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			n := ParseName(tc.input)
			got := n.DisplayShortest()
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNameFilepath(t *testing.T) {
	n := ParseName("llama3")
	fp := n.Filepath()
	got := strings.ReplaceAll(fp, string(filepath.Separator), "/")
	want := "registry.ollama.ai/library/llama3/latest"
	if got != want {
		t.Errorf("Filepath: got %q, want %q", got, want)
	}
}

func TestNameFilepath_PanicsWhenUnqualified(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unqualified name, got none")
		}
	}()
	n := Name{Model: "llama3"} // no host/namespace/tag
	_ = n.Filepath()
}

func TestMerge(t *testing.T) {
	a := Name{Model: "llama3", Tag: "latest"}
	b := DefaultName()
	got := Merge(a, b)
	if got.Host != defaultHost {
		t.Errorf("Host not filled from default: %q", got.Host)
	}
	if got.Model != "llama3" {
		t.Errorf("Model overwritten: %q", got.Model)
	}
}

func TestEqualFold(t *testing.T) {
	a := ParseName("Llama3")
	b := ParseName("llama3")
	if !a.EqualFold(b) {
		t.Error("EqualFold should be case-insensitive")
	}
}

func TestParseNameFromFilepath(t *testing.T) {
	cases := []struct {
		input   string
		wantFQ  bool
		wantStr string
	}{
		{
			input:   "registry.ollama.ai/library/llama3/latest",
			wantFQ:  true,
			wantStr: "registry.ollama.ai/library/llama3:latest",
		},
		{
			input:  "bad/path",
			wantFQ: false,
		},
		{
			input:  "too/many/parts/here/extra",
			wantFQ: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			// Use OS separator
			fp := strings.ReplaceAll(tc.input, "/", string(filepath.Separator))
			n := ParseNameFromFilepath(fp)
			if n.IsFullyQualified() != tc.wantFQ {
				t.Errorf("IsFullyQualified: got %v, want %v", n.IsFullyQualified(), tc.wantFQ)
			}
			if tc.wantFQ && n.String() != tc.wantStr {
				t.Errorf("String: got %q, want %q", n.String(), tc.wantStr)
			}
		})
	}
}
