package inspect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ollama-inspector/internal/fixtures"
)

var (
	configHex = strings.Repeat("a", 64)
	modelHex  = strings.Repeat("b", 64)
)

var (
	configContent = []byte(`{"architecture":"llama","model_format":"gguf","model_family":"llama","model_type":"8B","file_type":"Q4_0","context_length":8192,"os":"linux","rootfs":{"type":"layers","diff_ids":[]}}`)
	modelContent  = []byte("FAKE-GGUF-WEIGHTS")
)

// fakeStore builds a temporary OLLAMA_MODELS directory containing a single
// manifest for "testmodel:latest" (registry.ollama.ai/library/testmodel/latest)
// with a config blob and a model blob. It returns the store root; callers
// still need to point OLLAMA_MODELS at it via t.Setenv.
func fakeStore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	blobsDir := filepath.Join(root, "blobs")

	fixtures.MustMkdirAll(t, blobsDir)

	fixtures.MustWriteFile(t, fixtures.BlobPath(blobsDir, configHex), configContent, 0o644)
	fixtures.MustWriteFile(t, fixtures.BlobPath(blobsDir, modelHex), modelContent, 0o644)

	manifestDir := filepath.Join(root, "manifests", "registry.ollama.ai", "library", "testmodel")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.v2json",
		"config": map[string]any{
			"mediaType": "application/vnd.ollama.image.config",
			"digest":    "sha256:" + configHex,
			"size":      len(configContent),
		},
		"layers": []map[string]any{
			{
				"mediaType": "application/vnd.ollama.image.model",
				"digest":    "sha256:" + modelHex,
				"size":      len(modelContent),
			},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	fixtures.MustWriteFile(t, filepath.Join(manifestDir, "latest"), data, 0o644)

	return root
}

// fakeStoreWithParams is like fakeStore but also adds a params layer, for
// tests covering the params-in-report feature.
func fakeStoreWithParams(t *testing.T, paramsContent []byte) string {
	t.Helper()
	root := t.TempDir()

	blobsDir := filepath.Join(root, "blobs")
	fixtures.MustMkdirAll(t, blobsDir)

	fixtures.MustWriteFile(t, fixtures.BlobPath(blobsDir, configHex), configContent, 0o644)
	fixtures.MustWriteFile(t, fixtures.BlobPath(blobsDir, modelHex), modelContent, 0o644)

	paramsHex := strings.Repeat("f", 64)
	fixtures.MustWriteFile(t, fixtures.BlobPath(blobsDir, paramsHex), paramsContent, 0o644)

	manifestDir := filepath.Join(root, "manifests", "registry.ollama.ai", "library", "testmodel")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.v2json",
		"config": map[string]any{
			"mediaType": "application/vnd.ollama.image.config",
			"digest":    "sha256:" + configHex,
			"size":      len(configContent),
		},
		"layers": []map[string]any{
			{
				"mediaType": "application/vnd.ollama.image.model",
				"digest":    "sha256:" + modelHex,
				"size":      len(modelContent),
			},
			{
				"mediaType": "application/vnd.ollama.image.params",
				"digest":    "sha256:" + paramsHex,
				"size":      len(paramsContent),
			},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	fixtures.MustWriteFile(t, filepath.Join(manifestDir, "latest"), data, 0o644)

	return root
}

func TestRun_JSONOutput_IncludesParams(t *testing.T) {
	paramsContent := []byte(`{"temperature":0.7,"num_ctx":4096}`)
	root := fakeStoreWithParams(t, paramsContent)
	t.Setenv("OLLAMA_MODELS", root)

	done := fixtures.CaptureStdout(t)
	err := Run([]string{"testmodel"})
	out := done()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var report struct {
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, out)
	}
	if report.Params == nil {
		t.Fatalf("params field missing from JSON output:\n%s", out)
	}

	var params map[string]any
	if err := json.Unmarshal(report.Params, &params); err != nil {
		t.Fatalf("params field is not embedded as valid JSON: %v\nraw: %s", err, report.Params)
	}
	if params["temperature"] != 0.7 {
		t.Errorf("params.temperature = %v, want 0.7", params["temperature"])
	}
	if params["num_ctx"] != float64(4096) {
		t.Errorf("params.num_ctx = %v, want 4096", params["num_ctx"])
	}
}

func TestRun_JSONOutput_OmitsParamsWhenAbsent(t *testing.T) {
	root := fakeStore(t)
	t.Setenv("OLLAMA_MODELS", root)

	done := fixtures.CaptureStdout(t)
	err := Run([]string{"testmodel"})
	out := done()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, out)
	}
	if _, ok := raw["params"]; ok {
		t.Errorf("params field should be omitted when the model has no params layer, got %s", raw["params"])
	}
}

func TestRun_NoArgs(t *testing.T) {
	if err := Run(nil); err == nil {
		t.Fatal("expected error when no model name is given, got nil")
	}
}

func TestRun_TooManyArgs(t *testing.T) {
	if err := Run([]string{"a", "b"}); err == nil {
		t.Fatal("expected error for multiple positional args, got nil")
	}
}

func TestRun_InvalidModelName(t *testing.T) {
	err := Run([]string{"bad model name"}) // spaces are not a valid name character
	if err == nil {
		t.Fatal("expected error for invalid model name, got nil")
	}
	if !strings.Contains(err.Error(), "invalid or ambiguous model name") {
		t.Errorf("error = %v, want it to mention an invalid/ambiguous name", err)
	}
}

func TestRun_ManifestNotFound(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "manifests"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLLAMA_MODELS", root)

	err := Run([]string{"doesnotexist"})
	if err == nil {
		t.Fatal("expected error for a model with no manifest, got nil")
	}
	if !strings.Contains(err.Error(), "loading manifest") {
		t.Errorf("error = %v, want it to mention 'loading manifest'", err)
	}
}

func TestRun_JSONOutput(t *testing.T) {
	root := fakeStore(t)
	t.Setenv("OLLAMA_MODELS", root)

	done := fixtures.CaptureStdout(t)
	err := Run([]string{"testmodel"})
	out := done()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var report struct {
		ShortName string `json:"short_name"`
		Config    struct {
			Architecture string `json:"architecture"`
			ContextLen   int    `json:"context_length"`
		} `json:"config"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, out)
	}

	if report.ShortName != "testmodel:latest" {
		t.Errorf("short_name = %q, want %q", report.ShortName, "testmodel:latest")
	}
	if report.Config.Architecture != "llama" {
		t.Errorf("architecture = %q, want %q", report.Config.Architecture, "llama")
	}
	if report.Config.ContextLen != 8192 {
		t.Errorf("context_length = %d, want 8192", report.Config.ContextLen)
	}
}
