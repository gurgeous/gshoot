package ux

import (
	"io"
	"os"

	lipgloss "charm.land/lipgloss/v2"
)

// Println writes terminal output through lipgloss.
func Println(args ...any) {
	Fprintln(os.Stdout, args...)
}

// Printf writes formatted terminal output through lipgloss.
func Printf(format string, args ...any) {
	Fprintf(os.Stdout, format, args...)
}

// Fprint writes terminal output to w through lipgloss.
func Fprint(w io.Writer, args ...any) {
	_, _ = lipgloss.Fprint(w, args...)
}

// Fprintln writes terminal output to w through lipgloss.
func Fprintln(w io.Writer, args ...any) {
	_, _ = lipgloss.Fprintln(w, args...)
}

// Fprintf writes formatted terminal output to w through lipgloss.
func Fprintf(w io.Writer, format string, args ...any) {
	_, _ = lipgloss.Fprintf(w, format, args...)
}
