package cli

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/gurgeous/gshoot/ux"
)

//
// Main entrypoint
//

var (
	Version   = ""
	CommitSHA = ""
)

type app struct {
	stdout io.Writer
	stderr io.Writer
}

type CLI struct {
	Version kong.VersionFlag `short:"v" help:"Print the version number"`
	Auth    AuthCmd          `cmd:"" help:"Login or logout from Google Sheets."`
	List    ListCmd          `cmd:"" help:"List your Google Sheets."`
	Down    DownCmd          `cmd:"" help:"Download a Google Sheet as CSV."`
}

func Main(args []string, stdout, stderr io.Writer) int {
	// Version
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

	// init
	ux.Init()

	cli := &CLI{}
	ctx := kong.Parse(
		cli,
		kong.Name("gshoot"),
		kong.Description(fmt.Sprintf("Magically %s from Google Sheets.", ux.Brand.Render("import/export CSVs"))),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
		kong.Vars{
			"version":       version,
			"versionNumber": Version,
		},
		kong.Writers(stdout, stderr),
	)
	if err := ctx.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}
