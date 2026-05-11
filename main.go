package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/gurgeous/gshoot/commands"
	"github.com/gurgeous/gshoot/ux"
	// "github.com/k0kubun/pp/v3"
)

var (
	Version   = ""
	CommitSHA = ""
)

type CLI struct {
	Version kong.VersionFlag `short:"v" help:"Print the version number"`
	// Auth    commands.AuthCmd `cmd:"" help:"Login or logout from Google Sheets."`
	List commands.ListCmd `cmd:"" help:"List your Google Sheets."`
	Down commands.DownCmd `cmd:"" help:"Download a Google Sheet as CSV."`
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

	//
	// Kong (note that kong handles --help and --version internally)
	//

	// hack - no args becomes --help
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "--help")
	}

	cli := &CLI{}
	ctx := kong.Parse(
		cli,
		kong.Name("gshoot"),
		kong.Description(fmt.Sprintf("Magically %s from Google Sheets.", ux.Brand.Render("upload/download CSVs"))),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
		kong.Vars{
			"version":       version,
			"versionNumber": Version,
		},
	)

	if err := ctx.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
