package internal

import (
	"ollama-inspector/ollama"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BlobInfo aggregates information about a single blob file.
type BlobInfo struct {
	Digest    string
	Path      string
	Size      int64
	RefBy     []string // short model names that reference this blob
	MediaType string   // first known media type for this digest
}

func BuildBlobInfoIndex(blobsDir string, entries []os.DirEntry) map[string]*BlobInfo {
	blobs := make(map[string]*BlobInfo)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Blob filenames use "sha256-<hex64>"; reject anything else (temp files, etc.)
		if !strings.HasPrefix(name, "sha256-") || len(name) != len("sha256-")+64 {
			continue
		}
		digest := "sha256:" + name[len("sha256-"):]
		p := filepath.Join(blobsDir, name)
		fi, err := e.Info()
		if err != nil {
			continue
		}
		blobs[digest] = &BlobInfo{
			Digest: digest,
			Path:   p,
			Size:   fi.Size(),
		}
	}
	return blobs
}

func IndexBlobReferences(manifestsIndex map[ollama.Name]*ollama.Manifest, blobsIndex map[string]*BlobInfo) {
	for name, m := range manifestsIndex {
		short := name.DisplayShortest()

		// Note: deliberately NOT append(m.Layers, m.Config) here - append
		// can write into m.Layers's backing array when it has spare
		// capacity, silently corrupting the manifest's own layer slice
		// for any other code holding a reference to it in this run.
		seenDigests := make(map[string]bool, len(m.Layers)+1)
		record := func(l ollama.Layer) {
			if seenDigests[l.Digest] {
				return
			}
			seenDigests[l.Digest] = true
			if b, ok := blobsIndex[l.Digest]; ok {
				b.RefBy = append(b.RefBy, short)
				if b.MediaType == "" {
					b.MediaType = ollama.MediaTypeShortName(l.MediaType)
				}
			}
		}
		for _, l := range m.Layers {
			record(l)
		}
		record(m.Config)
	}

	// Map iteration order above is randomized, so without this the order
	// of names in RefBy (and therefore the "blobs" table output) would
	// vary from run to run.
	for _, b := range blobsIndex {
		sort.Strings(b.RefBy)
	}
}

func SortBlobs(blobsIndex map[string]*BlobInfo) []*BlobInfo {
	sorted := make([]*BlobInfo, 0, len(blobsIndex))
	for _, b := range blobsIndex {
		sorted = append(sorted, b)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Size > sorted[j].Size
	})
	return sorted
}
