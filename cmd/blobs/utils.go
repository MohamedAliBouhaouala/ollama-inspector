package blobs

import (
	"fmt"
	"ollama-inspector/internal"
	"path/filepath"
	"strings"
)

func printPathsOnly(rows []BlobRow) error {
	for _, r := range rows {
		fmt.Println(r.Path)
	}
	return nil
}

func buildRows(sorted []*internal.BlobInfo, orphansOnly bool) []BlobRow {
	rows := make([]BlobRow, 0, len(sorted))
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

		rows = append(rows, BlobRow{
			Digest: internal.TruncateDigest(b.Digest),
			Size:   internal.FmtBytes(b.Size),
			Type:   mt,
			RefBy:  refs,
			File:   filepath.Base(b.Path),
			Path:   b.Path,
			Raw:    b.Size,
		})
	}
	return rows
}

func formatRefs(refs []string) string {
	if len(refs) <= maxRefsShown {
		return strings.Join(refs, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(refs[:maxRefsShown], ", "), len(refs)-maxRefsShown)
}
