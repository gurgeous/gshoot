package sub

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/ux"
)

const helpHint = "gshoot: try 'gshoot --help' for more information"

func writeError(w io.Writer, err error) {
	if _, ok := errors.AsType[*auth.NoAuthError](err); ok {
		fmt.Fprintln(w, ux.Error.Render("You will need to authenticate first."))
		fmt.Fprintln(w)
		fmt.Fprintln(w, ux.Subtle.Render("I apologize in advance, setting up auth with Google Sheets is"))
		fmt.Fprintln(w, ux.Subtle.Render("annoyingly difficult for some reason. Don't blame gshoot."))
		fmt.Fprintln(w)
		fmt.Fprintln(w, ux.Subtle.Render("Try this first:"))
		fmt.Fprintln(w)
		fmt.Fprintln(w, ux.Info.Render("gshoot auth status"))
		return
	}
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
