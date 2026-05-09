package util

import (
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

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
func RPad(text string, padding int) string {
	spaces := max(padding-len(text), 1)
	return text + strings.Repeat(" ", spaces)
}

// Truncate shortens text to fit width and appends an ellipsis when needed.
func Truncate(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}

	var buf strings.Builder
	for _, r := range text {
		next := buf.String() + string(r)
		if lipgloss.Width(next) > width-1 {
			break
		}
		buf.WriteRune(r)
	}
	return buf.String() + "…"
}
