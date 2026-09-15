package blobs

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"ollama-inspector/internal"
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

	// Sort by size descending
	sorted := internal.SortBlobs(blobs)

	if quiet {
		return printPathsOnly(sorted, *orphansOnly)
	}

	fmt.Printf("Blobs directory: %s\n\n", blobsDir)

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "DIGEST\tSIZE\tTYPE\tREFERENCED BY\tFILE")

	count, totalSize := getCountAndSize(sorted, *orphansOnly, tw)

	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Printf("\n%d blob(s) — total %s\n", count, internal.FmtBytes(totalSize))
	return nil
}

func printPathsOnly(sorted []*internal.BlobInfo, orphansOnly bool) error {
	for _, b := range sorted {
		if orphansOnly && len(b.RefBy) > 0 {
			continue
		}
		fmt.Println(b.Path)
	}
	return nil
}

func getCountAndSize(sorted []*internal.BlobInfo, orphansOnly bool, tabWriter *tabwriter.Writer) (int, int64) {
	var totalSize int64
	count := 0
	for _, b := range sorted {
		if orphansOnly && len(b.RefBy) > 0 {
			continue
		}
		refs := "(orphan)"
		if len(b.RefBy) > 0 {
			refs = formatRefs(b.RefBy)

		}
		mt := b.MediaType
		if mt == "" {
			mt = "?"
		}

		shortDigest := internal.TruncateDigest(b.Digest)

		fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%s\n", shortDigest, internal.FmtBytes(b.Size),
			mt, refs, filepath.Base(b.Path))

		totalSize += b.Size
		count++
	}
	return count, totalSize
}

func formatRefs(refs []string) string {
	if len(refs) <= maxRefsShown {
		return strings.Join(refs, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(refs[:maxRefsShown], ", "), len(refs)-maxRefsShown)
}
