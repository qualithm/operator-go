package cli

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// newTable returns a tabwriter configured for the CLI's aligned columns.
func newTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
}

// dash renders empty strings as "-" so table columns never collapse.
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// row writes a tab-separated line to the tabwriter.
func row(tw *tabwriter.Writer, cols ...string) {
	for i, col := range cols {
		if i > 0 {
			_, _ = fmt.Fprint(tw, "\t")
		}
		_, _ = fmt.Fprint(tw, col)
	}
	_, _ = fmt.Fprintln(tw)
}
