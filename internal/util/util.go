package util

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

const (
	ESC = "\x1b"
	BEL = "\a"
	CSI = ESC + "["
	OSC = ESC + "]"
	ST  = ESC + "\\"
)

//
// terminal
//

// func Hyperlink(link, name string) string {

// Hyperlink returns an OSC 8 hyperlink on TTY output, else plain text.
func Hyperlink(w io.Writer, link, name string) string {
	if !IsTty(w) {
		return name
	}
	return OSC + "8;;" + link + ST + name + OSC + "8;;" + ST
}

// IsTty reports whether a writer is backed by a terminal.
func IsTty(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

//
// strings
//

// IndentBlock prefixes each line with two spaces.
func IndentBlock(text string) string {
	if text == "" {
		return ""
	}
	var buf strings.Builder
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("  ")
		buf.WriteString(line)
	}
	return buf.String()
}

// RPad right-pads text up to the requested width.
func RPad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds ", padding)
	return fmt.Sprintf(template, s)
}

// Truncate shortens text to fit width and appends an ellipsis when needed.
func Truncate(s string, length int) string {
	return ansi.Truncate(s, length, "…")
}
