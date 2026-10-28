package cli

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"
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

// when renders a time for a table: RFC 3339 in UTC, or "-" when unset.
func when(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

// parseExpiresAt reads the --expires-at flag as RFC 3339; an empty value is no expiry.
func parseExpiresAt(env Env, value string) (*time.Time, bool) {
	if value == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --expires-at must be an RFC 3339 time, like 2026-12-31T00:00:00Z")
		return nil, false
	}
	return &t, true
}

// nextPage tells the reader how to fetch the page after this one, when there is one.
func nextPage(w io.Writer, next *string) {
	if next != nil && *next != "" {
		_, _ = fmt.Fprintf(w, "\nmore: --cursor %s\n", *next)
	}
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
