package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gurgeous/gshoot/app"
	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/gmv"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

//
// Main entry point
//

type CLI struct {
	Version kong.VersionFlag `short:"v" help:"Print the version number"`
	Auth    AuthCmd          `cmd:"" help:"Login or logout from Google Sheets."`
	Down    DownCmd          `cmd:"" help:"Download a Google Sheet as CSV."`
	Up      UpCmd            `cmd:"" help:"Upload a CSV to Google Sheets."`
	List    ListCmd          `cmd:"" help:"List your Google Sheets."`
	Peek    PeekCmd          `cmd:"" help:"List sheets in a spreadsheet."`
	Wipe    WipeCmd          `cmd:"" help:"Wipe/delete all data from a spreadsheet."`
}

func Main(args []string, version string) error {
	//
	// Init real early, this sets up color styles
	//

	app.Init()

	//
	// welcome?
	//

	isFirstRun := !util.FileExists(util.ConfigDir())
	isNaked := len(args) == 0
	isWelcome := len(args) == 1 && args[0] == "welcome"

	if (isFirstRun && isNaked) || isWelcome {
		// show welcome movie, then auth status
		if app.Env.Smoke {
			fmt.Fprintln(os.Stdout, "welcome")
		} else {
			_ = gmv.Demo(context.Background())
		}
		return ShowAuthStatus()
	}

	//
	// Kong (note that kong handles --help and --version internally)
	//

	// fake --help when naked
	if isNaked {
		args = append(args, "--help")
	}

	parser := kong.Must(
		&CLI{},
		kong.Name("gshoot"),
		kong.Description("Magically upload/download CSVs from Google Sheets."),
		kong.Help(ux.HelpPrinter),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
		kong.Writers(os.Stdout, os.Stderr),
		kong.Vars{"version": version},
	)
	ctx, err := parser.Parse(args)
	if err != nil {
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) && parseErr.Context != nil {
			_ = parseErr.Context.PrintUsage(false)
			fmt.Fprintln(os.Stdout)
		}
		return err
	}

	//
	// all (non-auth) commands require login
	//

	if err := preflight(ctx); err != nil {
		return err
	}

	//
	// run the command
	//

	if err := ctx.Run(); err != nil {
		return err
	}

	return nil
}

//
// preflight - if not logged in, give some auth hints
//

func preflight(ctx *kong.Context) error {
	// auth commands don't need preflight
	if strings.HasPrefix(ctx.Command(), "auth") {
		return nil
	}

	// logged in?
	manager, err := auth.NewManager()
	if err != nil {
		return err
	}
	if manager.LoggedIn() {
		return nil
	}

	// not logged in - show status
	_ = ShowAuthStatus()

	// report error
	var msg string
	if manager.HasClientSecrets() {
		// botched browser flow
		msg = "you must complete `gshoot auth login` first"
	} else {
		// don't say "gshoot auth login" because of --client-secret
		msg = "you must authenticate first"
	}
	fmt.Fprintln(os.Stderr)
	return errors.New(msg)
}
