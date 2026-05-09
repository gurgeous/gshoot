package util

import "strings"

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
