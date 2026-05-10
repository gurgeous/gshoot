package util

import (
	"io"
	"os"
	"regexp"
	"strings"
	"time"

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

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

//
// strings
//

var indentRE = regexp.MustCompile(`(?m)^`)

func DateAndTimeStr(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Local().Format("Mon Jan 2 2006 15:04 MST")
}

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
	spaces := max(length-len(s), 1)
	return s + strings.Repeat(" ", spaces)
}

func SpreadsheetURL(id string) string {
	return "https://docs.google.com/spreadsheets/d/" + id
}

func Truncate(s string, length int) string {
	return ansi.Truncate(s, length, ellipsis)
}
