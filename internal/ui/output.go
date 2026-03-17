package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// Printer writes formatted output to a writer.
type Printer struct {
	Out io.Writer
	Err io.Writer
}

// DefaultPrinter writes to stdout/stderr.
var DefaultPrinter = &Printer{Out: os.Stdout, Err: os.Stderr}

// PrintSuccess prints a green success message.
func PrintSuccess(msg string, args ...any) {
	fmt.Fprintf(DefaultPrinter.Out, "%s %s\n", Success.Render("✓"), fmt.Sprintf(msg, args...))
}

// PrintError prints a red error message.
func PrintError(msg string, args ...any) {
	fmt.Fprintf(DefaultPrinter.Err, "%s %s\n", Error.Render("✗"), fmt.Sprintf(msg, args...))
}

// PrintWarning prints a yellow warning message.
func PrintWarning(msg string, args ...any) {
	fmt.Fprintf(DefaultPrinter.Err, "%s %s\n", Warning.Render("!"), fmt.Sprintf(msg, args...))
}

// PrintInfo prints a blue info message.
func PrintInfo(msg string, args ...any) {
	fmt.Fprintf(DefaultPrinter.Out, "%s %s\n", Info.Render("→"), fmt.Sprintf(msg, args...))
}

// PrintKeyValue prints a label-value pair.
func PrintKeyValue(label, value string) {
	fmt.Fprintf(DefaultPrinter.Out, "  %s %s\n", Label.Render(label+":"), value)
}

// PrintHeader prints a bold section header.
func PrintHeader(title string) {
	fmt.Fprintln(DefaultPrinter.Out, Bold.Render(title))
}

// PrintTable prints rows as an aligned table.
func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(DefaultPrinter.Out, 0, 0, 2, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(w, Dim.Render(strings.Join(headers, "\t")))
	}
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// CheckMark returns a green check or red cross.
func CheckMark(ok bool) string {
	if ok {
		return Success.Render("✓")
	}
	return Error.Render("✗")
}
