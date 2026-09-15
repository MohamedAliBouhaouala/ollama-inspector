package ollama

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Size returns the total size of all layers including config.
func (m *Manifest) Size() (size int64) {
	for _, layer := range append(m.Layers, m.Config) {
		size += layer.Size
	}
	return
}

// Digest returns the hex-encoded SHA-256 of the manifest file.
func (m *Manifest) Digest() string {
	return m.digest
}

// FileInfo returns the os.FileInfo of the manifest file.
func (m *Manifest) FileInfo() os.FileInfo {
	return m.fi
}

// Filepath returns the on-disk path to this manifest file.
func (m *Manifest) Filepath() string {
	return m.filepath
}

// ReadConfig reads and unmarshals the manifest's Config layer as ConfigV2.
func (m *Manifest) ReadConfig() (*ConfigV2, error) {
	p, err := BlobsPath(m.Config.Digest)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var cfg ConfigV2
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ParseNamedManifest loads and parses the manifest for the given fully-qualified name.
func ParseNamedManifest(n Name) (*Manifest, error) {
	if !n.IsFullyQualified() {
		return nil, Unqualified(n)
	}

	manifests, err := Path()
	if err != nil {
		return nil, err
	}

	p := filepath.Join(manifests, n.Filepath())

	var m Manifest
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.Warn("Failed to close manifest file", "path", p, "err", err)
		}
	}()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// Read the whole file up front rather than teeing it through the JSON
	// decoder. json.Decoder buffers internally and is not guaranteed to
	// pull every byte of the underlying reader through a single Read()
	// call, so a TeeReader placed in front of it can silently under-hash
	// the file (it happened to work here only because manifests are small
	// enough to fit in one buffered read). Hashing the fully-read byte
	// slice guarantees the digest matches `sha256sum <manifest file>`
	// regardless of file size or decoder buffering behavior.
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	sum := sha256.Sum256(data)

	m.filepath = p
	m.fi = fi
	m.digest = hex.EncodeToString(sum[:])

	return &m, nil
}

// Manifests enumerates all manifests in the local store.
func Manifests(continueOnError bool) (map[Name]*Manifest, error) {
	manifests, err := Path()
	if err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(filepath.Join(manifests, "*/*/*/*"))

	if err != nil {
		return nil, err
	}

	ms := make(map[Name]*Manifest)
	for _, match := range matches {
		fi, err := os.Stat(match)
		if err != nil {
			return nil, err
		}

		if !fi.IsDir() {
			rel, err := filepath.Rel(manifests, match)
			if err != nil {
				if !continueOnError {
					return nil, fmt.Errorf("%s %w", match, err)
				}
				slog.Warn("bad filepath", "path", match, "error", err)
				continue
			}

			n := ParseNameFromFilepath(rel)
			if !n.IsValid() {
				// Skip non-manifest files (e.g. editor swap files, .DS_Store).
				if !continueOnError {
					return nil, fmt.Errorf("%s %w", rel, err)
				}
				slog.Warn("bad manifest name", "path", rel)
				continue
			}

			m, err := ParseNamedManifest(n)
			if err != nil {
				if !continueOnError {
					return nil, fmt.Errorf("%s %w", n, err)
				}
				slog.Warn("bad manifest", "name", n, "error", err)
				continue
			}

			ms[n] = m
		}
	}

	return ms, nil
}
