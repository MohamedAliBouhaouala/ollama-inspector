// Package inspect implements the "inspect" sub-command.
// It resolves an Ollama manifest by name, loads the config blob,
// and prints a structured human-readable report.
package inspect

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"ollama-inspector/cmd/cli"
	"ollama-inspector/internal"
	"ollama-inspector/ollama"
)

const usage = `Usage: ollama-inspector inspect <model>

Inspect a locally stored Ollama model manifest.

Arguments:
  model   Model name, e.g. "llama3", "mistral:7b",
          "registry.ollama.ai/library/llama3:latest"

Flags:
`

// Run is the entry point for the inspect sub-command.
func Run(args []string) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage); fs.PrintDefaults() }

	positional, flagArgs := cli.SplitArgsAndFlags(args, map[string]bool{})

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	if len(positional) != 1 {
		fs.Usage()
		return fmt.Errorf("exactly one model name required")
	}

	n := ollama.ParseName(positional[0])
	if !n.IsFullyQualified() {
		return fmt.Errorf("invalid or ambiguous model name %q", positional[0])
	}

	m, err := ollama.ParseNamedManifest(n)
	if err != nil {
		// Provide a more actionable error when the models directory itself
		// cannot be located (common on fresh Linux system installs).
		if errors.Is(err, ollama.ErrModelsNotFound) {
			return fmt.Errorf(
				"could not find Ollama models directory.\n"+
					"  Set OLLAMA_MODELS to the correct path, e.g.:\n"+
					"    export OLLAMA_MODELS=/usr/share/ollama/.ollama/models\n"+
					"  Original error: %w", err)
		}
		return fmt.Errorf("loading manifest: %w", err)
	}

	// Build the high-level Model view; surface any non-fatal layer warnings.
	model, layerErrs := ollama.NewModelFromManifest(n, m)
	for _, e := range layerErrs {
		fmt.Fprintf(os.Stderr, "warning: %v\n", e)
	}

	return internal.FormatJSON(model, m)
}
