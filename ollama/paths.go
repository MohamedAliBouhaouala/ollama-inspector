package ollama

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// systemModelsPaths are the well-known Linux system-level Ollama model stores,
// used as a fallback when OLLAMA_MODELS is unset and no per-user store exists.
var systemModelsPaths = systemModelsPathsForGOOS(runtime.GOOS)

func systemModelsPathsForGOOS(goos string) []string {
	switch goos {
	case "linux":
		return []string{"/usr/share/ollama/.ollama/models"}
	default:
		return nil
	}
}

// Var returns a trimmed environment variable value.
func Var(key string) string {
	return strings.Trim(strings.TrimSpace(os.Getenv(key)), "\"'")
}

// Models returns the path to the models directory.
// Resolution order:
//  1. OLLAMA_MODELS environment variable (if set and non-empty)
//  2. $HOME/.ollama/models          (standard per-user install)
//  3. /usr/share/ollama/.ollama/models  (Linux system-wide install)
//
// An error is returned only when none of the candidates exist on disk.
func Models() (string, error) {
	if s := Var("OLLAMA_MODELS"); s != "" {
		return s, nil
	}

	var fallback string // first candidate that exists, even if empty

	tryCandidate := func(p string) (string, bool) {
		if _, err := os.Stat(p); err != nil {
			return "", false
		}
		if fallback == "" {
			fallback = p
		}
		return p, hasManifests(p)
	}
	for _, home := range candidateHomeDirs() {
		p := filepath.Join(home, ".ollama", "models")
		if hasManifests(p) {
			return p, nil
		}
		if found, ok := tryCandidate(p); ok {
			return found, nil
		}
	}

	// Fall back to system-wide paths (Linux package / Docker installs).
	for _, p := range systemModelsPaths {
		if hasManifests(p) {
			return p, nil
		}
		if found, ok := tryCandidate(p); ok {
			return found, nil
		}
	}

	// Nothing had manifests, but something existed — use it rather than
	// failing outright; ParseNamedManifest will give a clear "no such
	// manifest" error instead of a misleading "not found" here.
	if fallback != "" {
		return fallback, nil
	}

	return "", fmt.Errorf("%w: set OLLAMA_MODELS or install Ollama", ErrModelsNotFound)
}

// Path returns the manifests directory under the models root.
//
//	The directory is NOT created; this tool is read-only.
func Path() (string, error) {
	modelsPath, err := Models()
	if err != nil {
		return "", err
	}

	path := filepath.Join(modelsPath, "manifests")

	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("manifests dir not found at %s: %w", path, err)
	}

	return path, nil
}

func BlobsPath(digest string) (string, error) {
	if digest != "" && !digestPattern.MatchString(digest) {
		return "", ErrInvalidDigestFormat
	}

	digest = strings.ReplaceAll(digest, ":", "-")

	modelsPath, err := Models()
	if err != nil {
		return "", err
	}

	blobsDir := filepath.Join(modelsPath, "blobs")
	if digest == "" {
		return blobsDir, nil
	}
	return filepath.Join(blobsDir, digest), nil
}
