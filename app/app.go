package app

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
	"github.com/gurgeous/gshoot/env"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

// App owns process-wide config and I/O streams.
type App struct {
	Config env.Config // environment config

	// raw
	stdin  *os.File
	stdout io.Writer
	stderr io.Writer

	// potentially downsampled (eats ansi escapes)
	out *colorprofile.Writer
	err *colorprofile.Writer
}

// New initializes process-wide app state.
func New() *App {
	cfg := env.NewConfig()
	ux.Init(cfg)

	return &App{
		Config: cfg,
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		out:    colorprofile.NewWriter(os.Stdout, os.Environ()),
		err:    colorprofile.NewWriter(os.Stderr, os.Environ()),
	}
}

// Println writes stdout text through lipgloss.
func (a *App) Println(args ...any) {
	_, _ = fmt.Fprintln(a.out, args...)
}

// Printf writes formatted stdout text through lipgloss.
func (a *App) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(a.out, format, args...)
}

// Eprintln writes stderr text through lipgloss.
func (a *App) Eprintln(args ...any) {
	_, _ = fmt.Fprintln(a.err, args...)
}

// RawStdout returns stdout without lipgloss downsampling.
func (a *App) RawStdout() io.Writer {
	return a.stdout
}

// RawStderr returns stderr without lipgloss downsampling.
func (a *App) RawStderr() io.Writer {
	return a.stderr
}

// Hyperlink returns an OSC8 hyperlink when stdout is a TTY.
func (a *App) Hyperlink(link, name string) string {
	if !util.IsTty(a.stdout) {
		return name
	}
	return util.RenderHyperlink(link, name)
}

// Confirm asks a y/N question and exits when declined.
func (a *App) Confirm(prompt string) {
	_, _ = fmt.Fprintf(a.err, "%s %s ", ux.Warn.Render(prompt), ux.Muted.Render("(y/n)"))
	line, _ := bufio.NewReader(a.stdin).ReadString('\n')
	ok := len(line) > 0 && (line[0] == 'y' || line[0] == 'Y')
	if !ok {
		os.Exit(0)
	}
}

// Boom writes a fatal error banner to stderr.
func (a *App) Boom(msg string) {
	a.Eprintln(ux.Fatal.Render(fmt.Sprintf("gshoot: %-64s", msg)))
}

// Fatal writes a fatal error banner and exits.
func (a *App) Fatal(msg string) {
	a.Boom(msg)
	os.Exit(1)
}
