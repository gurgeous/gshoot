package commands

import (
	"bufio"
	"fmt"
	"os"

	cenv "github.com/caarlos0/env/v11"

	"github.com/charmbracelet/colorprofile"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

//
// App owns config and styled i/o
//

type App struct {
	// env
	Smoke bool   `env:"GSHOOT_SMOKE"` // use deterministic smoke-test behavior
	Theme string `env:"GSHOOT_THEME"` // force light or dark UI theme

	// styled streams, potentially downsampled
	Out *colorprofile.Writer `env:"-"`
	Err *colorprofile.Writer `env:"-"`
}

// NewApp initializes process-wide app state.
func NewApp() *App {
	// init env
	app, err := cenv.ParseAs[App]()
	if err != nil {
		app = App{}
	}

	// setup ux
	ux.Init(app.Theme)

	// now setup styled (downsampled) streams
	app.Out = colorprofile.NewWriter(os.Stdout, os.Environ())
	app.Err = colorprofile.NewWriter(os.Stderr, os.Environ())
	return &app
}

//
// styled output helpers
//

func (a *App) Println(args ...any) {
	_, _ = fmt.Fprintln(a.Out, args...)
}

func (a *App) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(a.Out, format, args...)
}

func (a *App) Eprintln(args ...any) {
	_, _ = fmt.Fprintln(a.Err, args...)
}

func (a *App) Eprintf(format string, args ...any) {
	_, _ = fmt.Fprintf(a.Err, format, args...)
}

//
// misc
//

// Hyperlink returns an OSC8 hyperlink when stdout is a TTY.
func (a *App) Hyperlink(link, name string) string {
	if !util.IsTty(os.Stdout) {
		return name
	}
	return util.RenderHyperlink(link, name)
}

//
// confirm/book/fatal
//

// Confirm asks a y/N question and exits when declined.
func (a *App) Confirm(prompt string) {
	_, _ = fmt.Fprintf(a.Err, "%s %s ", ux.Warn.Render(prompt), ux.Muted.Render("(y/n)"))
	if !util.IsTty(os.Stdin) {
		os.Exit(1)
	}

	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	ok := len(line) > 0 && (line[0] == 'y' || line[0] == 'Y')
	if !ok {
		os.Exit(1)
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
