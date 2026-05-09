package auth

import (
	"fmt"
	"io"

	"github.com/gurgeous/gshoot/internal/ux"
)

// WriteNoAuthError renders a richer, styled no-auth guidance card.
func WriteNoAuthError(w io.Writer, _ *NoAuthError) {
	fmt.Fprintln(w, ux.Error.Render("You will need to authenticate first."))
	fmt.Fprintln(w)

	fmt.Fprintln(w, ux.Subtle.Render("I apologize in advance, setting up auth with Google Sheets is"))
	fmt.Fprintln(w, ux.Subtle.Render("annoyingly difficult for some reason. Don't blame gshoot."))
	fmt.Fprintln(w)

	fmt.Fprintln(w, ux.Subtle.Render("Try this first:"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, ux.Info.Render("gshoot auth status"))
}
