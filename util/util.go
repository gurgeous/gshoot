package util

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/adrg/xdg"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

const (
	ESC      = "\x1b"
	CSI      = ESC + "["
	OSC      = ESC + "]"
	ST       = ESC + "\\"
	ellipsis = "…"
)

//
// terminal
//

// FileExists reports whether path exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// WritePrivateFile atomically writes data to path with 0600 permissions.
func WritePrivateFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// ConfigDir returns the gshoot config directory under XDG config home.
func ConfigDir() string {
	return filepath.Join(xdg.ConfigHome, "gshoot")
}

// RandomHex returns n random bytes encoded as lowercase hex.
func RandomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// OpenBrowserURL opens rawURL in the default browser for the current OS.
func OpenBrowserURL(url string) error {
	name, args := browserCommandArgs(runtime.GOOS, url)
	return exec.Command(name, args...).Start()
}

// Hyperlink returns an OSC8 hyperlink when the writer is a TTY.
func Hyperlink(w io.Writer, link, name string) string {
	if !IsTty(w) {
		return name
	}
	return OSC + "8;;" + link + ST + name + OSC + "8;;" + ST
}

// IsTty reports whether the writer wraps a terminal file descriptor.
func IsTty(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

// EnterRawMode enters stdin raw mode and switches stdout to alt screen.
func EnterRawMode() (func(), error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	fmt.Fprint(os.Stdout, ansi.SetModeAltScreenSaveCursor, ansi.EraseEntireScreen, ansi.HideCursor)

	cleanup := func() {
		fmt.Fprint(os.Stdout, ansi.ResetModeSynchronizedOutput)
		fmt.Fprint(os.Stdout, ansi.ResetStyle, ansi.ShowCursor, ansi.ResetModeAltScreenSaveCursor)
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}

	return cleanup, nil
}

// TerminalSize returns the current stdout size or the fallback.
func TerminalSize(fallback image.Point) image.Point {
	termW, termH, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fallback
	}
	return image.Pt(termW, termH)
}

//
// strings
//

var indentRE = regexp.MustCompile(`(?m)^`)

// DateAndTimeStr formats an RFC3339 timestamp in local time.
func DateAndTimeStr(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Local().Format("Mon Jan _2 2006 15:04 MST")
}

// DisplayWidth reports the rendered terminal width of a string.
func DisplayWidth(s string) int {
	return lipgloss.Width(s)
}

// Indent prefixes each line in s with indent.
func Indent(s string, indent string) string {
	if len(s) == 0 {
		return s
	}
	return indentRE.ReplaceAllLiteralString(s, indent)
}

// PadRight right-pads s, always adding at least one space.
func PadRight(s string, length int) string {
	spaces := max(length-len(s), 1)
	return s + strings.Repeat(" ", spaces)
}

// SpreadsheetURL builds a Google Sheets URL from an ID.
func SpreadsheetURL(id string) string {
	return "https://docs.google.com/spreadsheets/d/" + id
}

// Truncate trims s to the requested display width with an ellipsis.
func Truncate(s string, length int) string {
	return ansi.Truncate(s, length, ellipsis)
}

//
// csv
//

func CSVWrite(w io.Writer, rows [][]string) error {
	writer := csv.NewWriter(w)
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return nil
}

//
// helpers
//

func browserCommandArgs(goos, rawURL string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{rawURL}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", rawURL}
	default:
		return "xdg-open", []string{rawURL}
	}
}
