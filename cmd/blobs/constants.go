package blobs

const maxRefsShown = 3
const blobsTableHeader = "DIGEST\tSIZE\tTYPE\tREFERENCED BY\tFILE"
const defaultBlobsFormat = "table {{.Digest}}\t{{.Size}}\t{{.Type}}\t{{.RefBy}}\t{{.File}}"
