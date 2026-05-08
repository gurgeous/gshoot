package output

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
)

// DotsFrames is a compact spinner sequence suitable for inline progress.
var DotsFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Printer renders simple styled CLI output.
type Printer struct {
	stdout  io.Writer
	stderr  io.Writer
	info    lipgloss.Style
	success lipgloss.Style
	warn    lipgloss.Style
	failure lipgloss.Style
	subtle  lipgloss.Style
}

// New constructs a printer using Lip Gloss styling.
func New(stdout, stderr io.Writer) Printer {
	stdoutRenderer := lipgloss.NewRenderer(stdout)
	stderrRenderer := lipgloss.NewRenderer(stderr)

	return Printer{
		stdout:  stdout,
		stderr:  stderr,
		info:    stdoutRenderer.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		success: stdoutRenderer.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		warn:    stderrRenderer.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
		failure: stderrRenderer.NewStyle().Foreground(lipgloss.Color("204")).Bold(true),
		subtle:  stdoutRenderer.NewStyle().Foreground(lipgloss.Color("245")),
	}
}

func (p Printer) Info(msg string) {
	fmt.Fprintln(p.stdout, p.info.Render(msg))
}

func (p Printer) Success(msg string) {
	fmt.Fprintln(p.stdout, p.success.Render(msg))
}

func (p Printer) Subtle(msg string) {
	fmt.Fprintln(p.stdout, p.subtle.Render(msg))
}

func (p Printer) Warn(msg string) {
	fmt.Fprintln(p.stderr, p.warn.Render(msg))
}

func (p Printer) Error(msg string) {
	fmt.Fprintln(p.stderr, p.failure.Render(msg))
}

func (p Printer) SpinnerFrame(step int, msg string) string {
	frame := DotsFrames[step%len(DotsFrames)]
	return p.info.Render(frame + " " + msg)
}
