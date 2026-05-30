package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

type DownCmd struct {
	Output      string `short:"o" type:"path" help:"Where to write the CSV."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
	Sheet       string `arg:"" optional:"" name:"sheet" help:"Sheet name."`
}

func (c *DownCmd) Run() error {
	//
	// init
	//

	ctx := context.Background()
	dots := ux.StartDots(os.Stderr, "connecting to Google Sheets...")

	client, err := google.NewClient(ctx)
	if err != nil {
		return err
	}

	//
	// find spreadsheet
	//

	dots.SetDescription("finding spreadsheet...")
	spreadsheet, err := client.FindSpreadsheet(ctx, c.Spreadsheet)
	if err != nil {
		return fmt.Errorf("could not find spreadsheet '%s': %w", c.Spreadsheet, err)
	}
	if spreadsheet == nil {
		return fmt.Errorf("could not find spreadsheet '%s'", c.Spreadsheet)
	}

	//
	// find sheet
	//

	dots.SetDescription("finding specific sheet...")
	sheet, err := client.FindSheet(ctx, spreadsheet.ID, c.Sheet)
	if err != nil {
		return err
	}
	if sheet == nil {
		return fmt.Errorf("in spreadsheet '%s', could not find sheet '%s'", c.Spreadsheet, c.Sheet)
	}

	//
	// download
	//

	dots.SetDescription("downloading cells...")
	rows, err := client.GetRows(ctx, spreadsheet.ID, sheet.Title)
	if err != nil {
		return err
	}
	isStdout := c.Output == "" || c.Output == "-"
	if !isStdout {
		dots.SetDescription(fmt.Sprintf("saving %s", c.Output))
	}
	dots.Stop()

	//
	// write
	//

	writer := os.Stdout
	if !isStdout {
		file, err := os.Create(c.Output)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}
	return util.CSVWrite(writer, rows)
}
