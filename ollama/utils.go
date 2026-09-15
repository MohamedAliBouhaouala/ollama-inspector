package ollama

import (
	"os"
	"path/filepath"
)

// hasManifests reports whether p looks like a real Ollama models directory,
// i.e. it contains a non-empty "manifests" subdirectory. A directory that
// merely exists (e.g. an empty leftover, or a broken $HOME guess) is not
// good enough — we need one that actually holds data, otherwise a stray
// empty dir earlier in the search order would mask the real store.
func hasManifests(modelsDir string) bool {
	entries, err := os.ReadDir(filepath.Join(modelsDir, "manifests"))
	return err == nil && len(entries) > 0
}

// candidateHomeDirs returns home-directory candidates to try, guarding
// against a bogus $HOME. os.UserHomeDir() on Linux trusts $HOME verbatim
// (it does not consult /etc/passwd), so a misconfigured environment — e.g.
// running under sudo/a service with HOME unset or set to "/home" — can
// silently produce a directory like "/home/.ollama/models" that exists but
// is empty and is not the real store.
func candidateHomeDirs() []string {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, home)
	}
	return dirs
}

// MediaTypeShortName returns a human-readable label for a media type.
func MediaTypeShortName(mt string) string {
	switch mt {
	case MediaTypeImageModel:
		return "model"
	case MediaTypeImageParams:
		return "params"
	case MediaTypeImageSystem:
		return "system"
	case MediaTypeImageConfig:
		return "config"
	case MediaTypeImageJSON:
		return "json"
	case MediaTypeTemplate:
		return "template"
	case MediaTypeLicense:
		return "license"
	case MediaTypeAdapter:
		return "adapter"
	case MediaTypeProjector:
		return "projector"
	case MediaTypeImageTensor:
		return "tensor"
	default:
		return mt
	}
}
