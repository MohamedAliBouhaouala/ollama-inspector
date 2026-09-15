package ollama

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"ollama-inspector/internal/fixtures"
)

// buildManifestFile writes a manifest JSON file under
// root/manifests/<host>/<namespace>/<model>/<tag> and returns its raw bytes
// plus the Name pointing at it. extraLayers pads the manifest with filler
// layer entries so callers can exercise manifests larger than a single
// bufio/json.Decoder internal buffer (4096 bytes by default).
func buildManifestFile(t *testing.T, root string, extraLayers int) ([]byte, Name) {
	t.Helper()

	n := Name{Host: "registry.ollama.ai", Namespace: "library", Model: "digesttest", Tag: "latest"}

	configHex := strings.Repeat("a", 64)

	layers := []map[string]any{}
	for i := 0; i < extraLayers; i++ {
		layers = append(layers, map[string]any{
			"mediaType": "application/vnd.ollama.image.padding",
			"digest":    fmt.Sprintf("sha256:%064d", i),
			"size":      i,
		})
	}

	manifest := map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.v2json",
		"config": map[string]any{
			"mediaType": "application/vnd.ollama.image.config",
			"digest":    "sha256:" + configHex,
			"size":      10,
		},
		"layers": layers,
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	manifestDir := filepath.Join(root, "manifests", n.Host, n.Namespace, n.Model)
	fixtures.MustMkdirAll(t, manifestDir)
	fixtures.MustWriteFile(t, filepath.Join(manifestDir, n.Tag), data, 0o644)

	// The config blob's content is irrelevant to this test (only the
	// manifest file's own digest is under test), but it needs to exist
	// so unrelated code paths that stat/read it don't fail.
	blobsDir := filepath.Join(root, "blobs")
	fixtures.MustMkdirAll(t, blobsDir)
	fixtures.WriteBlob(t, blobsDir, configHex, []byte("0123456789"))

	return data, n
}

// TestParseNamedManifest_DigestMatchesFileSHA256 guards against a
// regression where the manifest digest was computed by tee-ing bytes
// through json.Decoder.Decode. That approach is not guaranteed to read the
// full file through a single Read() call, so the computed digest could
// silently diverge from `sha256sum <manifest file>` for larger manifests.
// Digest() must always equal the SHA-256 of the manifest's raw on-disk
// bytes, regardless of file size.
func TestParseNamedManifest_DigestMatchesFileSHA256(t *testing.T) {
	for _, tc := range []struct {
		name        string
		extraLayers int
	}{
		{"small manifest (fits in one decoder buffer read)", 0},
		{"large manifest (spans multiple reads)", 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("OLLAMA_MODELS", root)

			data, n := buildManifestFile(t, root, tc.extraLayers)

			want := sha256.Sum256(data)
			wantHex := hex.EncodeToString(want[:])

			m, err := ParseNamedManifest(n)
			if err != nil {
				t.Fatalf("ParseNamedManifest() error = %v", err)
			}

			if got := m.Digest(); got != wantHex {
				t.Errorf("Digest() = %s, want %s (sha256 of the raw manifest file)", got, wantHex)
			}

			// Sanity check the manifest was actually decoded correctly
			// (i.e. we're not passing by accident because parsing failed
			// silently and left a zero-value Manifest).
			if len(m.Layers) != tc.extraLayers {
				t.Errorf("len(Layers) = %d, want %d", len(m.Layers), tc.extraLayers)
			}
			if m.Config.Digest != "sha256:"+strings.Repeat("a", 64) {
				t.Errorf("Config.Digest = %s, want sha256:%s", m.Config.Digest, strings.Repeat("a", 64))
			}
		})
	}
}

// TestParseNamedManifest_DigestStableAcrossRepeatedParses ensures the
// digest is deterministic: parsing the same file twice must produce the
// same digest, catching any accidental reliance on read-order/buffering
// side effects.
func TestParseNamedManifest_DigestStableAcrossRepeatedParses(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OLLAMA_MODELS", root)

	_, n := buildManifestFile(t, root, 50)

	m1, err := ParseNamedManifest(n)
	if err != nil {
		t.Fatalf("ParseNamedManifest() error = %v", err)
	}
	m2, err := ParseNamedManifest(n)
	if err != nil {
		t.Fatalf("ParseNamedManifest() error = %v", err)
	}

	if m1.Digest() != m2.Digest() {
		t.Errorf("Digest() not stable across parses: %s vs %s", m1.Digest(), m2.Digest())
	}
}
