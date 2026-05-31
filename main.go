package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/commands"
	"github.com/gurgeous/gshoot/env"
	"github.com/gurgeous/gshoot/gmv"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

var (
	Version   = ""
	CommitSHA = ""
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

func main() {
	//
	// Version
	//

	if Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
			Version = info.Main.Version
		} else {
			Version = "built from source"
		}
	}
	version := fmt.Sprintf("gshoot: %s", Version)
	if len(CommitSHA) >= 7 {
		version += " (" + CommitSHA[:7] + ")"
	}

	//
	// init
	//

	ux.Init()
	envCfg := env.NewConfig()

	//
	// show welcome?
	//

	args := os.Args[1:]
	isFirstRun := !util.FileExists(util.ConfigDir())
	isNaked := len(args) == 0
	isWelcome := len(args) == 1 && args[0] == "welcome"

	if (isFirstRun && isNaked) || isWelcome {
		// show movie, then auth status
		if envCfg.Smoke {
			ux.Println("welcome")
		} else {
			_ = gmv.Demo(context.Background())
		}
		mustNewManager().ShowStatus()
		return
	}

	// fake --help when naked
	if isNaked {
		args = append(args, "--help")
	}

	//
	// Kong (note that kong handles --help and --version internally)
	//

	parser := kong.Must(
		&CLI{},
		kong.Name("gshoot"),
		kong.Description("Magically upload/download CSVs from Google Sheets."),
		kong.Help(ux.HelpPrinter),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
		kong.Vars{
			"version":       version,
			"versionNumber": Version,
		},
	)
	ctx, err := parser.Parse(args)
	if err != nil {
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) && parseErr.Context != nil {
			_ = parseErr.Context.PrintUsage(false)
			ux.Fprintln(os.Stdout)
		}
		fatal(err.Error())
	}

	//
	// preflight - all (non-auth) commands require login
	//

	if !strings.HasPrefix(ctx.Command(), "auth") {
		manager := mustNewManager()
		if !manager.LoggedIn() {
			var msg string
			if manager.HasClientSecrets() {
				// botched browser flow
				msg = "you must complete `gshoot auth login` first"
			} else {
				// don't say "gshoot auth login" because of --client-secret
				msg = "you must authenticate first"
			}
			boom(msg)
			ux.Fprintln(os.Stderr)
			manager.ShowStatus()
			os.Exit(1)
		}
	}

	//
	// run
	//

	if err := ctx.Run(); err != nil {
		fatal(err.Error())
	}
}

func mustNewManager() *auth.Manager {
	manager, err := auth.NewManager()
	if err != nil {
		fatal(err.Error())
	}
	return manager
}

func boom(msg string) {
	ux.Fprintln(os.Stderr, ux.Fatal.Render(fmt.Sprintf("gshoot: %-64s", msg)))
}

func fatal(msg string) {
	boom(msg)
	os.Exit(1)
}
