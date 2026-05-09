package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gurgeous/gshoot/internal/auth"
)

const helpHint = "gshoot: try 'gshoot --help' for more information"

func writeError(w io.Writer, err error) {
	if noAuth, ok := errors.AsType[*auth.NoAuthError](err); ok {
		auth.WriteNoAuthError(w, noAuth)
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
