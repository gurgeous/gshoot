package util

import (
	"io"
	"os"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

const (
	ESC      = "\x1b"
	BEL      = "\a"
	CSI      = ESC + "["
	OSC      = ESC + "]"
	ST       = ESC + "\\"
	ellipsis = "…"
)

//
// terminal
//

func Hyperlink(w io.Writer, link, name string) string {
	if !IsTty(w) {
		return name
	}
	return OSC + "8;;" + link + ST + name + OSC + "8;;" + ST
}

func IsTty(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

//
// strings
//

var indentRE = regexp.MustCompile(`(?m)^`)

func DisplayWidth(s string) int {
	return lipgloss.Width(s)
}

func Indent(s string, indent string) string {
	if len(s) == 0 {
		return s
	}
	return indentRE.ReplaceAllLiteralString(s, indent)
}

func PadRight(s string, length int) string {
	if padWidth := length - DisplayWidth(s); padWidth > 0 {
		s += strings.Repeat(" ", padWidth)
	}
	return s
}

func Truncate(s string, length int) string {
	return ansi.Truncate(s, length, ellipsis)
}
