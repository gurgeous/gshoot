package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/alecthomas/kong"
	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/ux"
)

var (
	version = "dev"
	tagline = fmt.Sprintf("Magically %s from Google Sheets.", ux.Brand.Render("import/export CSVs"))
)

type CLI struct {
	Version bool    `name:"version" short:"v" help:"Print version number."`
	Auth    AuthCmd `cmd:"" help:"Login or logout from Google Sheets."`
	List    ListCmd `cmd:"" help:"List your Google Sheets."`
	Down    DownCmd `cmd:"" help:"Download a Google Sheet as CSV."`
}

type app struct {
	stdout io.Writer
	stderr io.Writer
	ctx    *kong.Context
}

type exitCode int

func (c *CLI) Run(app *app) error {
	if c.Version {
		fmt.Fprintf(app.stdout, "gshoot %s\n", version)
		return nil
	}
	if app.ctx.Command() != "" {
		return nil
	}
	return app.ctx.PrintUsage(false)
}

func Main(args []string, stdout, stderr io.Writer) (code int) {
	ux.Init()

	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if exit, ok := recovered.(exitCode); ok {
			code = int(exit)
			return
		}
		panic(recovered)
	}()

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("gshoot"),
		kong.Description(tagline),
		kong.Writers(stdout, stderr),
		kong.Exit(func(code int) { panic(exitCode(code)) }),
	)
	if err != nil {
		writeError(stderr, err)
		return 1
	}

	ctx, err := parser.Parse(args)
	if err != nil {
		writeError(stderr, err)
		return 1
	}

	if err := ctx.Run(&app{stdout: stdout, stderr: stderr, ctx: ctx}); err != nil {
		writeError(stderr, err)
		return 1
	}

	return 0
}

type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Run browser OAuth login."`
	Logout AuthLogoutCmd `cmd:"" help:"Clear cached OAuth token."`
	Status AuthStatusCmd `cmd:"" help:"Show auth status."`
}

func (c *AuthCmd) Run(app *app) error {
	if app.ctx.Command() != "auth" {
		return nil
	}
	return app.ctx.PrintUsage(false)
}

type AuthLoginCmd struct {
	ClientSecretPath string `name:"client-secret" help:"Path to a downloaded Google Desktop app OAuth client JSON."`
}

func (c *AuthLoginCmd) Run(app *app) error {
	return runLogin(context.Background(), auth.LoginOptions{
		ClientSecretPath: c.ClientSecretPath,
		Stdout:           app.stdout,
		Stderr:           app.stderr,
	})
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(app *app) error {
	removed, err := runLogout()
	if err != nil {
		return err
	}
	if removed {
		fmt.Fprintln(app.stdout, "Removed cached OAuth token. OAuth client config was kept.")
		return nil
	}
	fmt.Fprintln(app.stdout, "No cached OAuth token was present.")
	return nil
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(app *app) error {
	writeStatus(app.stdout)
	return nil
}

type ListCmd struct{}

func (c *ListCmd) Run(app *app) error {
	ctx := context.Background()
	dots := ux.StartDots(app.stderr, "connecting to Google Sheets...")
	client, err := google.NewClient(ctx, google.ReadOnlyScopes())
	if err != nil {
		return err
	}

	dots.SetDescription("getting list of spreadsheets...")
	files, err := client.ListSpreadsheets(ctx, 10)
	if err != nil {
		return err
	}
	dots.SetDescription(fmt.Sprintf("%d recent spreadsheets", len(files)))
	dots.Stop()

	printFiles(app.stdout, files)
	return nil
}

type DownCmd struct {
	Output      string `name:"output" short:"o" help:"Where to write the CSV."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
	Sheet       string `arg:"" optional:"" name:"sheet" help:"Sheet name."`
}

func (c *DownCmd) Run(app *app) error {
	ctx := context.Background()
	dots := ux.StartDots(app.stderr, "connecting to Google Sheets...")
	defer dots.Stop()

	client, err := google.NewClient(ctx, google.ReadOnlyScopes())
	if err != nil {
		return err
	}

	dots.SetDescription("finding spreadsheet...")
	spreadsheet, err := client.FindSpreadsheet(ctx, c.Spreadsheet)
	if err != nil {
		return fmt.Errorf("could not find spreadsheet '%s'", c.Spreadsheet)
	}
	if spreadsheet == nil {
		return fmt.Errorf("could not find spreadsheet '%s'", c.Spreadsheet)
	}

	dots.SetDescription("finding specific sheet...")
	sheet, err := client.FindSheet(ctx, spreadsheet.Id, c.Sheet)
	if err != nil {
		return err
	}
	if sheet == nil {
		return fmt.Errorf("in spreadsheet '%s', could not find sheet '%s'", c.Spreadsheet, c.Sheet)
	}

	dots.SetDescription("downloading cells...")
	rows, err := client.GetRows(ctx, spreadsheet.Id, sheet)
	if err != nil {
		return err
	}
	if c.Output != "" {
		dots.SetDescription(fmt.Sprintf("saving %s", c.Output))
	}

	return writeRows(app.stdout, rows, c.Output)
}
