package util

import (
	"bufio"
	"cmp"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

//
// Small shared utils
//

const (
	ESC      = "\x1b"
	CSI      = ESC + "["
	OSC      = ESC + "]"
	ST       = ESC + "\\"
	ellipsis = "…"
)

// Clamp bounds v between lo and hi.
func Clamp[T cmp.Ordered](v, lo, hi T) T {
	return min(max(v, lo), hi)
}

//
// terminal
//

// SetCursorVisible shows or hides the terminal cursor.
func SetCursorVisible(w io.Writer, visible bool) {
	if !IsTty(w) {
		return
	}
	mode := ansi.ResetModeTextCursorEnable
	if visible {
		mode = ansi.SetModeTextCursorEnable
	}
	fmt.Fprint(w, mode)
}

// RenderHyperlink returns an OSC8 hyperlink string.
func RenderHyperlink(link, name string) string {
	return OSC + "8;;" + link + ST + name + OSC + "8;;" + ST
}

// Hyperlink returns an OSC8 hyperlink when stdout is a TTY.
func Hyperlink(link, name string) string {
	if !IsTty(os.Stdout) {
		return name
	}
	return RenderHyperlink(link, name)
}

// Confirm asks a y/N question and exits when declined.
func Confirm(prompt string) {
	_, _ = fmt.Fprint(os.Stderr, prompt, " ")
	if !IsTty(os.Stdin) {
		os.Exit(1)
	}

	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	ok := len(line) > 0 && (line[0] == 'y' || line[0] == 'Y')
	if !ok {
		os.Exit(1)
	}
}

// IsTty reports whether the writer wraps a terminal file descriptor.
func IsTty(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

//
// strings
//

// AllMatch reports whether every value matches re.
func AllMatch(values []string, re *regexp.Regexp) bool {
	for _, value := range values {
		if !re.MatchString(value) {
			return false
		}
	}
	return true
}

// AnyContains reports whether any value contains needle.
func AnyContains(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

// ContainsString reports whether values contains target.
func ContainsString(values []string, target string) bool {
	return IndexOfString(values, target) >= 0
}

// DateAndTimeStr formats an RFC3339 timestamp in local time.
func DateAndTimeStr(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Local().Format("Mon Jan _2 2006 15:04 MST")
}

// FormatInt formats n with comma group separators.
func FormatInt(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}

	var out strings.Builder
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	out.WriteString(s[:rem])
	for ii := rem; ii < len(s); ii += 3 {
		out.WriteByte(',')
		out.WriteString(s[ii : ii+3])
	}
	return out.String()
}

// IndexOfString returns the first index of target, or -1 when missing.
func IndexOfString(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

// Truncate trims s to the requested display width with an ellipsis.
func Truncate(s string, length int) string {
	return ansi.Truncate(s, length, ellipsis)
}

//
// csv
//

// CSVRead reads CSV rows and pads them to a rectangular shape.
func CSVRead(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("csv is empty: %s", path)
	}
	return CSVRectangularize(rows), nil
}

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

// CSVRectangularize pads rows so every row has the same column count.
func CSVRectangularize(rows [][]string) [][]string {
	cols := 0
	for _, row := range rows {
		cols = max(cols, len(row))
	}

	out := make([][]string, 0, len(rows))
	for _, src := range rows {
		dst := append([]string(nil), src...)
		if len(dst) < cols {
			dst = append(dst, make([]string, cols-len(dst))...)
		}
		out = append(out, dst)
	}
	return out
}

// CSVString renders rows as CSV text.
func CSVString(rows [][]string) string {
	var buf strings.Builder
	_ = CSVWrite(&buf, rows)
	return buf.String()
}

//
// misc
//

// OpenBrowserURL opens rawURL in the default browser for the current OS.
// There's no point to returning an error IMO, this can fail brutally on headless
// machines.
func OpenBrowserURL(url string) {
	name, args := browserCommandArgs(runtime.GOOS, url)
	_ = exec.Command(name, args...).Start()
}

// SpreadsheetURL builds a Google Sheets URL from an ID.
func SpreadsheetURL(id string) string {
	return "https://docs.google.com/spreadsheets/d/" + id
}

//
// numbers
//

// DecimalPrecision returns the max decimal precision, clamped to four places.
func DecimalPrecision(values []string) int {
	precision := 1
	for _, value := range values {
		if parts := strings.SplitN(value, ".", 2); len(parts) == 2 {
			precision = max(precision, len(parts[1]))
		}
	}
	return min(precision, 4)
}

//
// internal
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
