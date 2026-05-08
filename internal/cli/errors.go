package cli

import (
	"fmt"
	"io"
	"strings"
)

const helpHint = "gshoot: try 'gshoot --help' for more information"

func writeError(w io.Writer, err error) {
	fmt.Fprintf(w, "gshoot: %s\n%s\n", errorSummary(err), helpHint)
}

func errorSummary(err error) string {
	msg := strings.TrimSpace(err.Error())
	if line, _, ok := strings.Cut(msg, "\n"); ok {
		msg = line
	}
	if name, ok := cobraUnknownCommand(msg); ok {
		return fmt.Sprintf("unknown command %q", name)
	}
	return strings.TrimPrefix(msg, "gshoot: ")
}

func cobraUnknownCommand(msg string) (string, bool) {
	const prefix = `unknown command "`
	if !strings.HasPrefix(msg, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(msg, prefix)
	name, _, ok := strings.Cut(rest, `"`)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}

func spreadsheetNotFoundError(name string) error {
	return fmt.Errorf("spreadsheet not found: %s", name)
}

func sheetNotFoundError(spreadsheet, sheet string) error {
	return fmt.Errorf("sheet not found in %s: %s", spreadsheet, sheet)
}

func noSheetsError(spreadsheet string) error {
	return fmt.Errorf("spreadsheet has no sheets: %s", spreadsheet)
}
