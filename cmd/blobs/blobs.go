package blobs

import (
	"flag"
	"fmt"
	"os"

	"ollama-inspector/internal"
	"ollama-inspector/internal/format"
	"ollama-inspector/ollama"
)

const usage = `Usage: ollama-inspector blobs [flags]

List all blob files in the local Ollama model store.

Flags:
`

// Run is the entry point for the blobs sub-command.
func Run(args []string) error {
	fs := flag.NewFlagSet("blobs", flag.ContinueOnError)
	orphansOnly := fs.Bool("orphans", false, "Show only blobs not referenced by any manifest")
	var quiet bool
	fs.BoolVar(&quiet, "quiet", false, "Print only each blob's absolute path, one per line (for scripting)")
	fs.BoolVar(&quiet, "q", false, "Shorthand for -quiet")
	var formatFlag string
	fs.StringVar(&formatFlag, "format", "", "Go template for the output, e.g. '{{.Digest}}\\t{{.Size}}'; "+
		"prefix with \"table \" for aligned columns (this is the default)."+
		"Pass -format json for JSON output.")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage); fs.PrintDefaults() }

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Collect all blob files
	blobsDir, err := ollama.BlobsPath("")
	if err != nil {
		return fmt.Errorf("resolving blobs dir: %w", err)
	}

	entries, err := os.ReadDir(blobsDir)
	if err != nil {
		return fmt.Errorf("reading blobs dir %s: %w", blobsDir, err)
	}

	blobs := internal.BuildBlobInfoIndex(blobsDir, entries)

	// Cross-reference manifests
	ms, err := ollama.Manifests(true)
	if err != nil {
		return fmt.Errorf("reading manifests: %w", err)
	}

	internal.IndexBlobReferences(ms, blobs)

	// Sort by size descending, then filter down to what -orphans asked for.
	rows := buildRows(internal.SortBlobs(blobs), *orphansOnly)

	if quiet {
		return printPathsOnly(rows)
	}

	fmt.Printf("Blobs directory: %s\n\n", blobsDir)

	w, err := format.New(os.Stdout, formatFlag, defaultBlobsFormat)
	if err != nil {
		return err
	}

	if err := format.List(w, rows, blobsTableHeader); err != nil {
		return err
	}

	var totalSize int64
	for _, r := range rows {
		totalSize += r.Raw
	}
	fmt.Printf("\n%d blob(s) — total %s\n", len(rows), internal.FmtBytes(totalSize))

	return nil
}
