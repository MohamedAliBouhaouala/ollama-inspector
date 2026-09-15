package blobs

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"text/tabwriter"

	"ollama-inspector/internal"
	"ollama-inspector/internal/fixtures"
)

// ModelName is the short display name ("name:tag") of the model New creates.
const ModelName = "testmodel:latest"

var (
	configHex = strings.Repeat("a", 64)
	modelHex  = strings.Repeat("b", 64)
	orphanHex = strings.Repeat("c", 64)
)

var (
	configContent = []byte(`{"architecture":"llama","model_format":"gguf","model_type":"8B","file_type":"Q4_0","os":"linux","rootfs":{"type":"layers","diff_ids":[]}}`)
	modelContent  = []byte("FAKE-GGUF-WEIGHTS")
	orphanContent = []byte("UNREFERENCED-BLOB-DATA")
)

// fakeStore builds a temporary OLLAMA_MODELS directory containing:
//   - a manifest for "testmodel:latest" (registry.ollama.ai/library/testmodel/latest)
//     referencing a config blob and a model blob
//   - one extra blob that no manifest references (an orphan)
//
// It returns the store root; callers still need to point OLLAMA_MODELS at it
// via t.Setenv.

func fakeStore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	blobsDir := filepath.Join(root, "blobs")

	fixtures.MustMkdirAll(t, blobsDir)

	fixtures.WriteBlob(t, blobsDir, configHex, configContent)
	fixtures.WriteBlob(t, blobsDir, modelHex, modelContent)
	fixtures.WriteBlob(t, blobsDir, orphanHex, orphanContent)

	manifestDir := filepath.Join(root, "manifests", "registry.ollama.ai", "library", "testmodel")

	fixtures.MustMkdirAll(t, manifestDir)

	manifest := map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.v2+json",
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

func TestRun_BadFlag(t *testing.T) {
	// Unknown flag should fail during fs.Parse, before any filesystem access,
	// so no OLLAMA_MODELS setup is needed here.
	if err := Run([]string{"-nope"}); err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

func TestRun_MissingBlobsDir(t *testing.T) {
	root := t.TempDir() // exists, but has no "blobs" subdirectory
	t.Setenv("OLLAMA_MODELS", root)

	err := Run(nil)
	if err == nil {
		t.Fatal("expected error for missing blobs dir, got nil")
	}
	if !strings.Contains(err.Error(), "reading blobs dir") {
		t.Errorf("error = %v, want it to mention 'reading blobs dir'", err)
	}
}

func TestRun_ListsAllBlobs(t *testing.T) {
	root := fakeStore(t)
	t.Setenv("OLLAMA_MODELS", root)

	done := fixtures.CaptureStdout(t)
	err := Run(nil)
	out := done()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(out, "testmodel:latest") {
		t.Errorf("output missing reference to testmodel:latest:\n%s", out)
	}
	if !strings.Contains(out, "(orphan)") {
		t.Errorf("output missing orphan marker:\n%s", out)
	}
	if !strings.Contains(out, "3 blob(s)") {
		t.Errorf("expected all 3 blobs (config+model+orphan) counted:\n%s", out)
	}
}

func TestRun_ListsAllBlobs_PrintsDirectoryHeaderAndFilename(t *testing.T) {
	root := fakeStore(t)
	t.Setenv("OLLAMA_MODELS", root)

	done := fixtures.CaptureStdout(t)
	err := Run(nil)
	out := done()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	wantDir := filepath.Join(root, "blobs")

	if !strings.Contains(out, "Blobs directory: "+wantDir) {
		t.Errorf("output missing PATH column header:\n%s", out)
	}

	if !strings.Contains(out, "FILE") {
		t.Errorf("output missing FILE column header:\n%s", out)
	}

	if !strings.Contains(out, "sha256-"+configHex) {
		t.Errorf("output missing config blob filename:\n%s", out)
	}

	// The directory should only be printed once, not repeated per row.
	if n := strings.Count(out, wantDir); n != 1 {
		t.Errorf("blobs directory printed %d times, want exactly once:\n%s", n, out)
	}
}

func TestRun_QuietMode_PrintsOnlyAbsolutePaths(t *testing.T) {
	root := fakeStore(t)
	t.Setenv("OLLAMA_MODELS", root)

	done := fixtures.CaptureStdout(t)
	err := Run([]string{"-quiet"})
	out := done()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (one per blob):\n%s", len(lines), out)
	}

	wantConfigPath := filepath.Join(root, "blobs", "sha256-"+configHex)
	found := false
	for _, l := range lines {
		if l == wantConfigPath {
			found = true
		}
	}
	if !found {
		t.Errorf("quiet output missing config blob path %q:\n%s", wantConfigPath, out)
	}

	for _, unwanted := range []string{"Blobs directory", "DIGEST", "blob(s)", "REFERENCED BY"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("quiet output should contain only bare paths, found %q:\n%s", unwanted, out)
		}
	}
}

func TestRun_QuietShorthand_MatchesQuiet(t *testing.T) {
	root := fakeStore(t)
	t.Setenv("OLLAMA_MODELS", root)

	done := fixtures.CaptureStdout(t)
	err := Run([]string{"-q", "-orphans"})
	out := done()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	wantOrphanPath := filepath.Join(root, "blobs", "sha256-"+orphanHex)
	if strings.TrimSpace(out) != wantOrphanPath {
		t.Errorf("got %q, want exactly the orphan path %q", out, wantOrphanPath)
	}
}

func TestFormatRefs_UnderCapJoinsAll(t *testing.T) {
	got := formatRefs([]string{"a:latest", "b:latest"})
	want := "a:latest, b:latest"
	if got != want {
		t.Errorf("formatRefs() = %q, want %q", got, want)
	}
}

func TestFormatRefs_OverCapCollapsesRemainder(t *testing.T) {
	got := formatRefs([]string{"a:latest", "b:latest", "c:latest", "d:latest", "e:latest"})
	want := "a:latest, b:latest, c:latest (+2 more)"
	if got != want {
		t.Errorf("formatRefs() = %q, want %q", got, want)
	}
}

func TestRun_OrphansOnly(t *testing.T) {
	root := fakeStore(t)
	t.Setenv("OLLAMA_MODELS", root)

	done := fixtures.CaptureStdout(t)
	err := Run([]string{"-orphans"})
	out := done()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(out, "(orphan)") {
		t.Errorf("output missing orphan marker:\n%s", out)
	}
	if strings.Contains(out, "testmodel:latest") {
		t.Errorf("-orphans output should not list blobs referenced by testmodel:\n%s", out)
	}
	if !strings.Contains(out, "1 blob(s)") {
		t.Errorf("expected exactly 1 orphan blob:\n%s", out)
	}
}

func TestGetCountAndSize_AllBlobs(t *testing.T) {
	blobs := []*internal.BlobInfo{
		{Digest: "sha256:" + strings.Repeat("a", 64),
			Path: "/models/blobs/sha256-" + strings.Repeat("a", 64),
			Size: 100, RefBy: []string{"model-a:latest"}, MediaType: "config"},
		{Digest: "sha256:" + strings.Repeat("b", 64), Size: 200}, // orphan, no MediaType set
	}
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	orphansOnly := false
	count, total := getCountAndSize(blobs, orphansOnly, tw)
	if err := tw.Flush(); err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if total != 300 {
		t.Errorf("total = %d, want 300", total)
	}

	out := buf.String()

	if !strings.Contains(out, "model-a:latest") {
		t.Errorf("expected referencing model name in output:\n%s", out)
	}

	if !strings.Contains(out, "(orphan)") {
		t.Errorf("expected orphan marker for unreferenced blob:\n%s", out)
	}

	if !strings.Contains(out, "?") {
		t.Errorf("expected '?' fallback for empty media type:\n%s", out)
	}

	if !strings.Contains(out, "sha256-"+strings.Repeat("a", 64)) {
		t.Errorf("expected the blob's filename in output:\n%s", out)
	}

}

func TestGetCountAndSize_OrphansOnlyFiltersReferenced(t *testing.T) {
	blobs := []*internal.BlobInfo{
		{Digest: "sha256:" + strings.Repeat("a", 64), Size: 100, RefBy: []string{"model-a:latest"}},
		{Digest: "sha256:" + strings.Repeat("b", 64), Size: 200}, // orphan
	}
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	orphansOnly := true
	count, total := getCountAndSize(blobs, orphansOnly, tw)
	if err := tw.Flush(); err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if total != 200 {
		t.Errorf("total = %d, want 200", total)
	}
	if strings.Contains(buf.String(), "model-a:latest") {
		t.Errorf("orphans-only should exclude referenced blobs:\n%s", buf.String())
	}
}

func TestGetCountAndSize_Empty(t *testing.T) {
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	orphansOnly := false
	count, total := getCountAndSize(nil, orphansOnly, tw)
	if err := tw.Flush(); err != nil {
		t.Fatal(err)
	}

	if count != 0 || total != 0 {
		t.Errorf("count=%d total=%d, want 0,0 for empty input", count, total)
	}
}
