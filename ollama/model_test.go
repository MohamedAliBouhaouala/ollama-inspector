package ollama

import (
	"path/filepath"
	"strings"
	"testing"

	"ollama-inspector/internal/fixtures"
)

func writeBlobFile(t *testing.T, blobsDir, hexDigest string, content []byte) {
	t.Helper()
	fixtures.MustMkdirAll(t, blobsDir)
	fixtures.WriteBlob(t, blobsDir, hexDigest, content)
}

func TestNewModelFromManifest_ReadsParams(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OLLAMA_MODELS", root)
	blobsDir := filepath.Join(root, "blobs")

	configHex := strings.Repeat("a", 64)
	paramsHex := strings.Repeat("b", 64)
	paramsContent := []byte(`{"temperature":0.8,"num_ctx":8192,"stop":["<|eot_id|>"]}`)

	writeBlobFile(t, blobsDir, configHex, []byte(`{"architecture":"llama"}`))
	writeBlobFile(t, blobsDir, paramsHex, paramsContent)

	m := &Manifest{
		Config: Layer{MediaType: MediaTypeImageConfig, Digest: "sha256:" + configHex, Size: 24},
		Layers: []Layer{
			{MediaType: MediaTypeImageParams, Digest: "sha256:" + paramsHex, Size: int64(len(paramsContent))},
		},
	}

	model, errs := NewModelFromManifest(ParseName("testmodel"), m)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if model.Params != string(paramsContent) {
		t.Errorf("Params = %q, want %q", model.Params, string(paramsContent))
	}
}

func TestNewModelFromManifest_NoParamsLayerLeavesParamsEmpty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OLLAMA_MODELS", root)
	blobsDir := filepath.Join(root, "blobs")

	configHex := strings.Repeat("c", 64)
	writeBlobFile(t, blobsDir, configHex, []byte(`{"architecture":"llama"}`))

	m := &Manifest{
		Config: Layer{MediaType: MediaTypeImageConfig, Digest: "sha256:" + configHex, Size: 24},
	}

	model, errs := NewModelFromManifest(ParseName("testmodel"), m)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if model.Params != "" {
		t.Errorf("Params = %q, want empty when manifest has no params layer", model.Params)
	}
}

// TestNewModelFromManifest_MissingParamsBlobIsNonFatal matches the existing
// behavior for System/License layers: a missing/unreadable blob is reported
// as a warning in the returned []error, but does not abort assembly of the
// rest of the Model.
func TestNewModelFromManifest_MissingParamsBlobIsNonFatal(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OLLAMA_MODELS", root)
	blobsDir := filepath.Join(root, "blobs")

	configHex := strings.Repeat("d", 64)
	writeBlobFile(t, blobsDir, configHex, []byte(`{"architecture":"llama"}`))

	missingHex := strings.Repeat("e", 64) // no blob written for this digest
	m := &Manifest{
		Config: Layer{MediaType: MediaTypeImageConfig, Digest: "sha256:" + configHex, Size: 24},
		Layers: []Layer{
			{MediaType: MediaTypeImageParams, Digest: "sha256:" + missingHex, Size: 10},
		},
	}

	model, errs := NewModelFromManifest(ParseName("testmodel"), m)
	if len(errs) == 0 {
		t.Fatal("expected a non-fatal error for a missing params blob, got none")
	}
	if model.Params != "" {
		t.Errorf("Params = %q, want empty after a failed read", model.Params)
	}
	if model.Config.Architecture != "llama" {
		t.Errorf("rest of Model should still be assembled; Config.Architecture = %q, want %q",
			model.Config.Architecture, "llama")
	}
}
