package ollama

import (
	"errors"
	"io"
	"os"
)

// Open returns a ReadSeekCloser for the layer's blob.
func (l *Layer) Open() (io.ReadSeekCloser, error) {
	if l.Digest == "" {
		return nil, errors.New("opening layer with empty digest")
	}

	blob, err := BlobsPath(l.Digest)
	if err != nil {
		return nil, err
	}

	return os.Open(blob)
}

// BlobPath returns the filesystem path of this layer's blob.
func (l *Layer) BlobPath() (string, error) {
	if l.Digest == "" {
		return "", errors.New("layer has empty digest")
	}
	return BlobsPath(l.Digest)
}

// Exists reports whether the blob for this layer is present on disk.
func (l *Layer) Exists() bool {
	if l.Digest == "" {
		return false
	}
	p, err := BlobsPath(l.Digest)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}
