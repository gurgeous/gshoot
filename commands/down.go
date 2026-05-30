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
	rows, err := c.run0()
	if err != nil {
		return err
	}

	writer := os.Stdout
	if c.Output != "" && c.Output != "-" {
		file, err := os.Create(c.Output)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}
	return util.CSVWrite(writer, rows)
}

func (c *DownCmd) run0() (google.Rows, error) {
	//
	// init
	//

	ctx := context.Background()
	dots := ux.StartDots(os.Stderr, "connecting to Google Sheets...")
	defer dots.Stop()

	client, err := google.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	//
	// find spreadsheet file
	//

	dots.SetDescription("finding spreadsheet file...")
	spreadsheet, err := client.FindSpreadsheetFile(ctx, c.Spreadsheet)
	if err != nil {
		return nil, err
	}
	if spreadsheet == nil {
		return nil, fmt.Errorf("could not find spreadsheet file named '%s'", c.Spreadsheet)
	}

	//
	// find sheet
	//

	dots.SetDescription("finding sheet...")
	sheet, err := client.FindSheet(ctx, spreadsheet.ID, c.Sheet)
	if err != nil {
		return nil, err
	}
	if sheet == nil {
		return nil, fmt.Errorf("found spreadsheet file '%s', but could not find sheet named '%s'", c.Spreadsheet, c.Sheet)
	}

	//
	// download
	//

	dots.SetDescription("downloading rows...")
	rows, err := client.GetRows(ctx, spreadsheet.ID, sheet.Title)
	if err != nil {
		return nil, err
	}
	isStdout := c.Output == "" || c.Output == "-"
	if !isStdout {
		dots.SetDescription(fmt.Sprintf("saving %d rows to %s...", len(rows), c.Output))
	}
	return rows, nil
}
