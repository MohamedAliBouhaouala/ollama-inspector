package fixtures

import (
	"path/filepath"
	"testing"
)

// BlobPath returns the on-disk path of the blob created by New,
// for tests asserting against printed blob paths (e.g. inspect -v).
func BlobPath(dir string, hex string) string {
	return filepath.Join(dir, "sha256-"+hex)
}

func WriteBlob(t *testing.T, dir string, hexDigest string, content []byte) {
	t.Helper()
	p := BlobPath(dir, hexDigest)
	MustWriteFile(t, p, content, 0o644)

}
