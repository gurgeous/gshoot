package auth

import (
	"fmt"
	"io"

	"github.com/gurgeous/gshoot/internal/output"
)

// WriteNoAuthError renders a richer, styled no-auth guidance card.
func WriteNoAuthError(w io.Writer, err *NoAuthError) {
	ui := output.New(w, w)
	ui.Error("You will need to authenticate first.")
	fmt.Fprintln(w)

	ui.Subtle("I apologize in advance, setting up auth with Google Sheets is")
	ui.Subtle("annoyingly difficult for some reason. Don't blame gshoot.")
	fmt.Fprintln(w)

	ui.Subtle("Try this first:")
	fmt.Fprintln(w)
	ui.Info("gshoot auth status")
}
