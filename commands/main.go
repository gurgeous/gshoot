package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/gurgeous/gshoot/app"
	"github.com/gurgeous/gshoot/ux"
)

//
// Main entry point
//

type CLI struct {
	Version   kong.VersionFlag `short:"v" help:"Print the version number"`
	Append    AppendCmd        `cmd:"" help:"Append a CSV to an existing Google Sheet."`
	Down      DownCmd          `cmd:"" aliases:"d" help:"Download a Google Sheet as CSV."`
	Hyperlink HyperlinkCmd     `cmd:"" help:"Transform plaintext cells into hyperlinks."`
	Join      JoinCmd          `cmd:"" help:"Join a CSV into an existing Google Sheet."`
	Up        UpCmd            `cmd:"" aliases:"u" help:"Upload a CSV to Google Sheets."`
	List      ListCmd          `cmd:"" aliases:"ls" help:"List your Google Sheets."`
	Peek      PeekCmd          `cmd:"" help:"List sheets in a spreadsheet."`
	Wipe      WipeCmd          `cmd:"" help:"Wipe/delete all data from a spreadsheet."`
}

func Main(args []string, version string) error {
	//
	// Init real early, this sets up color styles
	//

	app.Init()

	isNaked := len(args) == 0

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
			fmt.Println()
		}
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
