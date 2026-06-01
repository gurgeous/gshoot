package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gurgeous/gshoot/app"
	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/commands"
	"github.com/gurgeous/gshoot/gmv"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

//
// Main CLI entrypoint
//

var (
	// from goreleaser
	version = ""
	commit  = ""
	date    = ""
)

type CLI struct {
	Version kong.VersionFlag `short:"v" help:"Print the version number"`
	Auth    commands.AuthCmd `cmd:"" help:"Login or logout from Google Sheets."`
	Down    commands.DownCmd `cmd:"" help:"Download a Google Sheet as CSV."`
	Up      commands.UpCmd   `cmd:"" help:"Upload a CSV to Google Sheets."`
	List    commands.ListCmd `cmd:"" help:"List your Google Sheets."`
	Peek    commands.PeekCmd `cmd:"" help:"List sheets in a spreadsheet."`
	Wipe    commands.WipeCmd `cmd:"" help:"Wipe/delete all data from a spreadsheet."`
}

// tiny wrapper around main0, with err handling
func main() {
	err := main0()
	if err != nil {
		fmt.Fprintln(os.Stderr, ux.Fatal.Render(fmt.Sprintf("gshoot: %-64s", err.Error())))
		os.Exit(1)
	}
}

func main0() error {
	//
	// Init real early, this sets up color styles
	//

	app.Init()

	//
	// welcome?
	//

	args := os.Args[1:]
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
		return commands.ShowAuthStatus()
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
		kong.Vars{
			"version": versionString(),
		},
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
	_ = commands.ShowAuthStatus()

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

// pull version string, either populated by gorelease or from debug.ReadBuildInfo
func versionString() string {
	modified := false

	if version == "" {
		version = "built from source"
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					commit = setting.Value
				case "vcs.time":
					date = setting.Value
				case "vcs.modified":
					if setting.Value == "true" {
						modified = true
					}
				}
			}
		}
	}

	c := commit[:7]
	if modified {
		c += "*"
	}

	return fmt.Sprintf("ghoot %s (%s, %s)", version, c, date[:16])
}
