package cli

import (
	"fmt"
	"io"

	"github.com/alecthomas/kong"
	"github.com/gurgeous/gshoot/ux"
)

var (
	Version   = "dev"
	CommitSHA = ""
)

//
// Kong
//

type CLI struct {
	Version bool    `name:"version" short:"v" help:"Print version number."`
	Auth    AuthCmd `cmd:"" help:"Login or logout from Google Sheets."`
	List    ListCmd `cmd:"" help:"List your Google Sheets."`
	Down    DownCmd `cmd:"" help:"Download a Google Sheet as CSV."`
}

//
// Main entrypoint
//

type app struct {
	stdout io.Writer
	stderr io.Writer
}

func Main(args []string, stdout, stderr io.Writer) (code int) {
	type exitCode int

	// init
	ux.Init()
	cli := CLI{}
	tagline := fmt.Sprintf("Magically %s from Google Sheets.", ux.Brand.Render("import/export CSVs"))
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

	// parse args
	ctx, err := parser.Parse(args)
	if err != nil {
		writeError(stderr, err)
		return 1
	}

	if cli.Version {
		fmt.Fprintf(stdout, "gshoot %s\n", Version)
		return 0
	}
	if ctx.Command() == "" || ctx.Command() == "auth" {
		if err := ctx.PrintUsage(false); err != nil {
			writeError(stderr, err)
			return 1
		}
		return 0
	}

	if err := ctx.Run(&app{stdout: stdout, stderr: stderr}); err != nil {
		writeError(stderr, err)
		return 1
	}

	return 0
}
