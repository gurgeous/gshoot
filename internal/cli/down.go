package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
)

type DownCmd struct {
	Output      string `name:"output" short:"o" help:"Where to write the CSV."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
	Sheet       string `arg:"" optional:"" name:"sheet" help:"Sheet name."`
}

func (c *DownCmd) Run(app *app) error {
	ctx := context.Background()
	dots := ux.StartDots(app.stderr, "connecting to Google Sheets...")

	client, err := google.NewClient(ctx, google.ReadOnlyScopes())
	if err != nil {
		return err
	}

	dots.SetDescription("finding spreadsheet...")
	spreadsheet, err := client.FindSpreadsheet(ctx, c.Spreadsheet)
	if err != nil {
		return fmt.Errorf("could not find spreadsheet '%s': %w", c.Spreadsheet, err)
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
	dots.Stop()

	return writeRows(app.stdout, rows, c.Output)
}

func writeRows(stdout io.Writer, rows google.Rows, outputPath string) error {
	writer := stdout
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}
	return util.CSVWrite(writer, rows)
}
