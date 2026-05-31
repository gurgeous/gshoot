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

	stdin  io.Reader            // raw stdin
	stdout io.Writer            // raw stdout
	stderr io.Writer            // raw stderr
	out    *colorprofile.Writer // styled stdout
	err    *colorprofile.Writer // styled stderr
}

// New initializes process-wide app state.
func New() *App {
	return NewWithIO(os.Stdin, os.Stdout, os.Stderr, env.NewConfig())
}

// NewWithWriters initializes app state for explicit output streams.
func NewWithWriters(stdout, stderr io.Writer, cfg env.Config) *App {
	return NewWithIO(os.Stdin, stdout, stderr, cfg)
}

// NewWithIO initializes app state for explicit input and output streams.
func NewWithIO(stdin io.Reader, stdout, stderr io.Writer, cfg env.Config) *App {
	initIn, initOut := os.Stdin, os.Stdout
	if in, ok := stdin.(*os.File); ok {
		initIn = in
	}
	if out, ok := stdout.(*os.File); ok {
		initOut = out
	}
	ux.Init(cfg, initIn, initOut)

	return &App{
		Config: cfg,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		out:    colorprofile.NewWriter(stdout, os.Environ()),
		err:    colorprofile.NewWriter(stderr, os.Environ()),
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
