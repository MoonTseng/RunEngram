package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Format describes how to render structured output to stdout.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

// Resolve picks a format from an explicit flag (may be ""), defaulting to JSON
// when stdout is not a TTY and table when it is.
func Resolve(flagVal string) Format {
	switch strings.ToLower(flagVal) {
	case "json":
		return FormatJSON
	case "table":
		return FormatTable
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return FormatTable
	}
	return FormatJSON
}

// Render writes v in the requested format. For table format the caller
// supplies a fallback rendering function (since tables are domain-specific).
func Render(w io.Writer, f Format, v any, table func(io.Writer)) error {
	if f == FormatTable && table != nil {
		table(w)
		return nil
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// JSON is a shortcut to write v as indented JSON.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Errf writes an error line to stderr.
func Errf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}
