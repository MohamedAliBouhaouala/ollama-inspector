package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ollama-inspector/internal/fixtures"
	"ollama-inspector/ollama"
)

// makeFakeBlobDir creates a temp dir with fake blob files and junk files.
func makeFakeBlobDir(t *testing.T) (string, []string) {
	t.Helper()
	dir := t.TempDir()

	// Valid blob names: sha256- + 64 hex chars
	validNames := []string{
		"sha256-" + "a" + rep('b', 63),
		"sha256-" + rep('c', 64),
	}
	for _, name := range validNames {

		fixtures.MustWriteFile(t, filepath.Join(dir, name), []byte("data"), 0o644)
	}
	// Junk: temp files, wrong prefix, wrong length
	junk := []string{
		"sha256-.tmp",
		"notsha256-" + rep('a', 64),
		"sha256-" + rep('a', 63), // too short
		"sha256-" + rep('a', 65), // too long
		".DS_Store",
	}

	for _, name := range junk {
		fixtures.MustWriteFile(t, filepath.Join(dir, name), []byte("junk"), 0o644)
	}

	return dir, validNames
}

func rep(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func TestBuildBlobInfoIndex_FiltersJunk(t *testing.T) {
	dir, validNames := makeFakeBlobDir(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	index := BuildBlobInfoIndex(dir, entries)

	if len(index) != len(validNames) {
		t.Errorf("got %d blobs, want %d", len(index), len(validNames))
	}

	for _, name := range validNames {
		digest := "sha256:" + name[len("sha256-"):]
		if _, ok := index[digest]; !ok {
			t.Errorf("missing expected digest %q", digest)
		}
	}
}

func TestBuildBlobInfoIndex_DigestFormat(t *testing.T) {
	dir := t.TempDir()
	hex64 := rep('a', 64)
	name := "sha256-" + hex64

	fixtures.MustWriteFile(t, filepath.Join(dir, name), []byte("x"), 0o644)

	entries, _ := os.ReadDir(dir)
	index := BuildBlobInfoIndex(dir, entries)

	want := "sha256:" + hex64
	b, ok := index[want]
	if !ok {
		t.Fatalf("digest %q not in index", want)
	}
	if b.Path != filepath.Join(dir, name) {
		t.Errorf("Path: got %q, want %q", b.Path, filepath.Join(dir, name))
	}
	if b.Size != 1 {
		t.Errorf("Size: got %d, want 1", b.Size)
	}
}

func TestIndexBlobReferences_DedupesSameDigestWithinOneManifest(t *testing.T) {
	// Reproduces the reported bug: a single manifest lists the same
	// content-addressed digest via two different layers (e.g. two
	// byte-identical license layers). The referencing model must appear
	// only once in RefBy, not once per layer that happens to share the
	// digest.
	licenseDigest := "sha256:" + rep('1', 64)
	modelDigest := "sha256:" + rep('2', 64)

	n := mustParseName(t, "qwen3-vl:2b")
	manifestsIndex := map[ollama.Name]*ollama.Manifest{
		n: {
			Config: ollama.Layer{MediaType: ollama.MediaTypeImageConfig, Digest: modelDigest},
			Layers: []ollama.Layer{
				{MediaType: ollama.MediaTypeLicense, Digest: licenseDigest},
				// A second, distinct license layer that happens to be
				// byte-identical to the first and therefore shares its
				// digest - this is what produced the duplicate name.
				{MediaType: ollama.MediaTypeLicense, Digest: licenseDigest},
				{MediaType: ollama.MediaTypeImageModel, Digest: modelDigest},
			},
		},
	}

	blobsIndex := map[string]*BlobInfo{
		licenseDigest: {Digest: licenseDigest},
		modelDigest:   {Digest: modelDigest},
	}

	IndexBlobReferences(manifestsIndex, blobsIndex)

	if got := blobsIndex[licenseDigest].RefBy; len(got) != 1 {
		t.Errorf("license blob RefBy = %v, want exactly one entry (got %d)", got, len(got))
	}
	if got := blobsIndex[modelDigest].RefBy; len(got) != 1 {
		t.Errorf("model blob RefBy = %v, want exactly one entry (got %d)", got, len(got))
	}
}

func TestIndexBlobReferences_SameBlobSharedAcrossModelsKeepsBothNames(t *testing.T) {
	// The fix must only dedupe *within* a single manifest. A blob that's
	// genuinely shared across two different models should still list
	// both of them.
	sharedDigest := "sha256:" + rep('3', 64)

	n1 := mustParseName(t, "qwen3-vl:2b")
	n2 := mustParseName(t, "qwen3-vl:4b")
	manifestsIndex := map[ollama.Name]*ollama.Manifest{
		n1: {Layers: []ollama.Layer{{MediaType: ollama.MediaTypeLicense, Digest: sharedDigest}}},
		n2: {Layers: []ollama.Layer{{MediaType: ollama.MediaTypeLicense, Digest: sharedDigest}}},
	}
	blobsIndex := map[string]*BlobInfo{sharedDigest: {Digest: sharedDigest}}

	IndexBlobReferences(manifestsIndex, blobsIndex)

	got := blobsIndex[sharedDigest].RefBy
	if len(got) != 2 {
		t.Fatalf("RefBy = %v, want 2 distinct model names", got)
	}
	want := []string{"qwen3-vl:2b", "qwen3-vl:4b"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("RefBy = %v, want %v (sorted)", got, want)
			break
		}
	}
}

func TestIndexBlobReferences_RefByIsSortedForManyReferencingModels(t *testing.T) {
	// Regression guard: RefBy must be sorted regardless of how many
	// manifests reference the blob or what order Go's map iteration
	// (which is intentionally randomized) happens to visit them in.
	// Without sorting, this test - and real "blobs" output - would be
	// flaky/nondeterministic from run to run.
	digest := "sha256:" + rep('6', 64)
	names := []string{"zebra:latest", "apple:latest", "mango:7b", "banana:1b"}

	manifestsIndex := make(map[ollama.Name]*ollama.Manifest, len(names))
	for _, s := range names {
		manifestsIndex[mustParseName(t, s)] = &ollama.Manifest{
			Layers: []ollama.Layer{{MediaType: ollama.MediaTypeLicense, Digest: digest}},
		}
	}
	blobsIndex := map[string]*BlobInfo{digest: {Digest: digest}}

	for i := 0; i < 20; i++ {
		blobsIndex[digest].RefBy = nil
		IndexBlobReferences(manifestsIndex, blobsIndex)

		got := blobsIndex[digest].RefBy
		want := []string{"apple:latest", "banana:1b", "mango:7b", "zebra:latest"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: RefBy = %v, want %v (sorted)", i, got, want)
		}
	}
}

func TestIndexBlobReferences_DoesNotMutateManifestLayers(t *testing.T) {
	// Regression guard for the append(m.Layers, m.Config) aliasing bug:
	// indexing references must never grow/overwrite the manifest's own
	// Layers slice.
	digest := "sha256:" + rep('4', 64)
	n := mustParseName(t, "testmodel:latest")

	layers := make([]ollama.Layer, 1, 4) // spare capacity, like append() would exploit
	layers[0] = ollama.Layer{MediaType: ollama.MediaTypeImageModel, Digest: digest}

	m := &ollama.Manifest{
		Config: ollama.Layer{MediaType: ollama.MediaTypeImageConfig, Digest: "sha256:" + rep('5', 64)},
		Layers: layers,
	}
	manifestsIndex := map[ollama.Name]*ollama.Manifest{n: m}
	blobsIndex := map[string]*BlobInfo{digest: {Digest: digest}}

	IndexBlobReferences(manifestsIndex, blobsIndex)

	if len(m.Layers) != 1 {
		t.Fatalf("m.Layers length changed: got %d, want 1 (IndexBlobReferences must not mutate it)", len(m.Layers))
	}
	if m.Layers[0].Digest != digest {
		t.Errorf("m.Layers[0] corrupted: got %+v", m.Layers[0])
	}
}

func mustParseName(t *testing.T, s string) ollama.Name {
	t.Helper()
	return ollama.ParseName(s)
}

func TestSortBlobs_BySize(t *testing.T) {
	index := map[string]*BlobInfo{
		"sha256:aaa": {Digest: "sha256:aaa", Size: 100},
		"sha256:bbb": {Digest: "sha256:bbb", Size: 5000},
		"sha256:ccc": {Digest: "sha256:ccc", Size: 300},
	}
	sorted := SortBlobs(index)
	if len(sorted) != 3 {
		t.Fatalf("got %d, want 3", len(sorted))
	}
	// Must be descending
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Size > sorted[i-1].Size {
			t.Errorf("not sorted descending at index %d: %d > %d", i, sorted[i].Size, sorted[i-1].Size)
		}
	}
	if sorted[0].Digest != "sha256:bbb" {
		t.Errorf("largest blob should be first, got %q", sorted[0].Digest)
	}
}
