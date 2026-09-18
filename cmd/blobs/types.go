package blobs

type BlobRow struct {
	Digest string
	Size   string
	Type   string
	RefBy  string
	File   string
	Path   string
	Raw    int64
}
