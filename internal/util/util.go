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

// does this file exist?
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// OSC8 hyperlink
func Hyperlink(w io.Writer, link, name string) string {
	if !IsTty(w) {
		return name
	}
	return OSC + "8;;" + link + ST + name + OSC + "8;;" + ST
}

// check if a writer (with underyling file) is a tty
func IsTty(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

//
// strings
//

var indentRE = regexp.MustCompile(`(?m)^`)

// convert time string to nicely formatted str
func DateAndTimeStr(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Local().Format("Mon Jan 2 2006 15:04 MST")
}

// measure term display width for a string
func DisplayWidth(s string) int {
	return lipgloss.Width(s)
}

// indend string and contained newlines
func Indent(s string, indent string) string {
	if len(s) == 0 {
		return s
	}
	return indentRE.ReplaceAllLiteralString(s, indent)
}

// pad string to the right, always using at last one space
func PadRight(s string, length int) string {
	spaces := max(length-len(s), 1)
	return s + strings.Repeat(" ", spaces)
}

// conver spreadsheet id to url
func SpreadsheetURL(id string) string {
	return "https://docs.google.com/spreadsheets/d/" + id
}

// truncate a string with ellipsis
func Truncate(s string, length int) string {
	return ansi.Truncate(s, length, ellipsis)
}
