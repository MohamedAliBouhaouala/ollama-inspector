package fixtures

import (
	"io"
	"log/slog"
	"os"
	"testing"
)

// CaptureStdout redirects os.Stdout for the duration of a test and returns a
// function that restores it and yields everything that was written.
func CaptureStdout(t *testing.T) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	return func() string {
		if closeErr := w.Close(); closeErr != nil {
			slog.Warn("failed to close stdout capture writer", "err", closeErr)
		}
		os.Stdout = old
		data, _ := io.ReadAll(r)
		return string(data)
	}
}

func MustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func MustWriteFile(t *testing.T, dir string, data []byte, perm os.FileMode) {
	if err := os.WriteFile(dir, data, perm); err != nil {
		t.Fatal(err)
	}
}
