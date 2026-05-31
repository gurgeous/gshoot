package commands

import (
	"errors"

	"github.com/gurgeous/gshoot/app"
	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
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
func (c *UpCmd) Run(a *app.App) (err error) {
	if c.Refill && c.Replace {
		return errors.New("use either --refill or --replace")
	}

	rows, err := util.CSVRead(c.CSVPath)
	if err != nil {
		return err
	}

	//
	// init
	//

	cmd, err := srunStart(srunOptions{spreadsheet: c.Spreadsheet, create: true})
	if err != nil {
		return err
	}
	defer func() { cmd.stop(err) }()

	//
	// upload
	//

	file, err := c.upload(cmd, google.Rows(rows))
	if err != nil {
		return err
	}

	// print url and maybe open
	url := util.SpreadsheetURL(file.ID) + "/edit"
	a.Println(url)
	if c.Open {
		util.OpenBrowserURL(url)
	}
	return nil
}

// upload runs the complete upload workflow.
func (c *UpCmd) upload(cmd *srun, rows google.Rows) (*google.File, error) {
	var err error

	//
	// get Spreadsheet for that File
	//

	cmd.dots.SayFetchSpreadsheet(cmd.file.Name)
	spreadsheet, err := cmd.client.GetSpreadsheet(cmd.ctx, cmd.file.ID)
	if err != nil {
		return nil, err
	}

	//
	// find/create target sheet
	//

	cmd.dots.SayUploadRows(len(rows), cmd.file.Name, c.Sheet)
	s := newUploader(cmd.ctx, cmd.client, cmd.file, spreadsheet, c, rows)
	s.id, err = s.resolveTargetSheet()
	if err != nil {
		return nil, err
	}

	//
	// --refill
	//

	var refill *refiller
	if c.Refill {
		refill, err = s.prepareRefiller()
		if err != nil {
			return nil, err
		}
	}

	pipeline := []struct {
		on  bool
		run func() error
	}{
		// paste
		{c.Replace, s.clearSheet}, // --replace
		{true, s.growSheet},       // add padding
		{true, s.pasteCSV},        // paste local csv

		// extend
		{c.Refill, func() error { return refill.extend() }}, // --refill

		// post-paste stuff
		{c.Filter, s.applyFilter},   // --filter
		{c.Numeric, s.applyNumeric}, // --numeric
		{c.Layout, s.applyLayout},   // --layout
	}
	for _, p := range pipeline {
		if p.on {
			if err := p.run(); err != nil {
				return nil, err
			}
		}
	}

	//
	// success!
	//

	return cmd.file, nil
}
