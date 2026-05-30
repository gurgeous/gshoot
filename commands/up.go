package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

// UpCmd uploads a CSV to Google Sheets.
type UpCmd struct {
	Sheet       string `help:"Destination sheet name."`
	Refill      bool   `help:"Merge CSV data INTO the sheet."`
	Replace     bool   `help:"Create or overwrite the destination sheet."`
	Filter      bool   `help:"Add a standard Google Sheets filter."`
	Layout      bool   `help:"Auto-size column width to fit cells."`
	Numeric     bool   `help:"Format obvious numeric columns."`
	Open        bool   `help:"Open the sheet URL when done."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
	CSVPath     string `arg:"" name:"csv" type:"path" help:"CSV file to upload."`
}

// Run uploads the configured CSV.
func (c *UpCmd) Run() error {
	if c.Refill && c.Replace {
		return errors.New("use either --refill or --replace")
	}

	//
	// read csv
	//

	dots := ux.StartDots(os.Stderr, "reading csv...")
	defer dots.Stop()

	rows, err := util.CSVRead(c.CSVPath)
	if err != nil {
		return err
	}

	//
	// init
	//

	dots.SetDescription("connecting to Google Sheets...")
	ctx := context.Background()
	client, err := google.NewClient(ctx)
	if err != nil {
		return err
	}

	//
	// upload
	//

	file, err := c.upload(ctx, client, dots, google.Rows(rows))
	if err != nil {
		return err
	}

	// print url and maybe open
	url := util.SpreadsheetURL(file.ID) + "/edit"
	fmt.Println(url)
	if c.Open {
		util.OpenBrowserURL(url)
	}
	return nil
}

// upload runs the complete upload workflow.
func (c *UpCmd) upload(ctx context.Context, client *google.Client, dots *ux.Dots, rows google.Rows) (*google.File, error) {
	var err error

	//
	// find/create File
	//

	dots.SetDescription("finding spreadsheet file...")
	file, err := client.FindSpreadsheet(ctx, c.Spreadsheet)
	if err != nil {
		return nil, err
	}
	if file == nil {
		dots.SetDescription(fmt.Sprintf("creating new spreadsheet '%s'...", c.Spreadsheet))
		file, err = client.CreateSpreadsheet(ctx, c.Spreadsheet)
		if err != nil {
			return nil, err
		}
	}

	//
	// get Spreadsheet for that File
	//

	dots.SetDescription("fetching spreadsheet metadata...")
	spreadsheet, err := client.GetSpreadsheet(ctx, file.ID)
	if err != nil {
		return nil, err
	}

	//
	// find/create sheet
	//

	dots.SetDescription(fmt.Sprintf("uploading %d rows to file '%s', sheet '%s'...", len(rows), file.Name, c.Sheet))
	sheet := newUploadSheet(ctx, client, file.ID, spreadsheet, c, rows)
	if err := sheet.ensure(); err != nil {
		return nil, err
	}

	//
	// --refill
	//

	var refill *sheetRefill
	if c.Refill {
		refill, err = newSheetRefill(sheet)
		if err != nil {
			return nil, err
		}
		sheet.rows, err = refill.mergedRows()
		if err != nil {
			return nil, err
		}
	}

	//
	// --replace
	//

	if c.Replace {
		if err := sheet.clear(); err != nil {
			return nil, err
		}
	}

	//
	// apply various flags
	//

	if err := sheet.resize(); err != nil {
		return nil, err
	}
	if err := sheet.paste(); err != nil {
		return nil, err
	}
	if c.Refill {
		if err := refill.extend(); err != nil {
			return nil, err
		}
	}
	if err := sheet.applyOptions(); err != nil {
		return nil, err
	}

	//
	// success!
	//

	return file, nil
}
