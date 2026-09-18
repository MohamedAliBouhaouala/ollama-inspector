package format

import (
	"io"
	"text/template"
)

// Writer renders values to an underlying io.Writer according to a format
// string. Construct one with New; it is not safe for concurrent use.
type Writer struct {
	out   io.Writer
	json  bool
	table bool
	tmpl  *template.Template
}
