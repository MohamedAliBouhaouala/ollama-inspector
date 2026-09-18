package format

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"text/template"
)

var funcMap = template.FuncMap{
	"join": strings.Join,
}

// New parses format and returns a Writer that renders to out. An empty
// format falls back to def, so callers always pass their command's default
// format string as def rather than special-casing "no -format flag given"
// at every call site.
func New(out io.Writer, format, def string) (*Writer, error) {
	if format == "" {
		format = def
	}
	if format == JSON {
		return &Writer{out: out, json: true}, nil
	}

	table := strings.HasPrefix(format, tablePrefix)
	src := strings.TrimPrefix(format, tablePrefix)

	tmpl, err := template.New("format").Funcs(funcMap).Parse(src)
	if err != nil {
		return nil, fmt.Errorf("parsing format %q: %w", format, err)
	}
	return &Writer{out: out, table: table, tmpl: tmpl}, nil
}

// One renders a single record, such as the result of `inspect`.
func (w *Writer) One(v any) error {
	if w.json {
		return encodeJSON(w.out, v)
	}
	if err := w.tmpl.Execute(w.out, v); err != nil {
		return fmt.Errorf("executing format template: %w", err)
	}
	fmt.Fprintln(w.out)
	return nil
}

// List renders items one record at a time. In table mode, columns are
// aligned with text/tabwriter and header (if non-empty) is written first as
// the tabwriter header row; header is ignored outside table mode, matching
// `docker ps --format '{{.ID}}'` printing bare values with no header.
func List[T any](w *Writer, items []T, header string) error {
	if w.json {
		return encodeJSON(w.out, items)
	}

	dst := w.out
	var tw *tabwriter.Writer
	if w.table {
		tw = tabwriter.NewWriter(w.out, 0, 4, 2, ' ', 0)
		dst = tw
		if header != "" {
			fmt.Fprintln(tw, header)
		}
	}

	for _, item := range items {
		if err := w.tmpl.Execute(dst, item); err != nil {
			return fmt.Errorf("executing format template: %w", err)
		}
		fmt.Fprintln(dst)
	}

	if tw != nil {
		return tw.Flush()
	}
	return nil
}

func encodeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
