package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func FormatParams(raw string) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(raw), "", "  "); err != nil {
		return strings.TrimSpace(raw)
	}
	return buf.String()
}

func FmtBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func TruncateDigest(digest string) string {
	if len(digest) > len("sha256:")+12 {
		return digest[:len("sha256:")+12] + "…"
	}
	return digest
}
