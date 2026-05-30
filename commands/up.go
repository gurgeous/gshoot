package commands

import (
	"context"
	"encoding/csv"
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

	rows, err := c.readCSV()
	if err != nil {
		return err
	}

	ctx := context.Background()
	dots := ux.StartDots(os.Stderr, "connecting to Google Sheets...")

	client, err := google.NewClient(ctx)
	if err != nil {
		return err
	}

	runner := &upRunner{
		cmd:    c,
		ctx:    ctx,
		client: client,
		rows:   rows,
	}
	if err := runner.upload(dots); err != nil {
		return err
	}

	dots.Stop()
	url := util.SpreadsheetURL(runner.file.ID) + "/edit"
	fmt.Println(url)
	if c.Open {
		util.OpenBrowserURL(url)
	}
	return nil
}

// REVIEW: add a suffix comment for each ivar
type upRunner struct {
	cmd         *UpCmd              // parsed CLI options
	ctx         context.Context     // upload request context
	client      *google.Client      // Google API client
	file        *google.File        // target spreadsheet file
	spreadsheet *google.Spreadsheet // target spreadsheet metadata
	rows        google.Rows         // CSV rows from disk
	sheet       *uploadSheet        // target sheet mutator
}

// upload runs the complete upload workflow.
func (u *upRunner) upload(dots *ux.Dots) error {
	if err := u.findOrCreateFile(dots); err != nil {
		return err
	}
	if err := u.loadSpreadsheet(); err != nil {
		return err
	}

	u.sheet = newUploadSheet(u.ctx, u.client, u.file.ID, u.spreadsheet, u.cmd, u.rows)

	dots.SetDescription(fmt.Sprintf("uploading %d rows to file '%s', sheet '%s'...", len(u.rows), u.file.Name, u.sheet.title))
	if err := u.sheet.ensure(u.cmd); err != nil {
		return err
	}

	uploadRows := u.rows
	var refill *refillUpload
	if u.cmd.Refill {
		refillData, err := newRefillUpload(u.ctx, u.client, u.file.ID, u.sheet.id, u.sheet.title, u.rows)
		if err != nil {
			return err
		}
		merged, err := refillData.rows()
		if err != nil {
			return err
		}
		refill = refillData
		uploadRows = merged
	}
	u.sheet.rows = uploadRows

	if u.cmd.Replace {
		if err := u.sheet.clear(); err != nil {
			return err
		}
	}
	if err := u.sheet.resize(); err != nil {
		return err
	}
	if err := u.sheet.paste(); err != nil {
		return err
	}
	if refill != nil {
		if err := refill.apply(u.sheet); err != nil {
			return err
		}
	}
	return u.sheet.applyOptions(u.cmd)
}

// REVIEW: move this to google client
// findOrCreateFile finds the target spreadsheet or creates it.
func (u *upRunner) findOrCreateFile(dots *ux.Dots) error {
	dots.SetDescription("finding spreadsheet...")
	file, err := u.client.FindSpreadsheet(u.ctx, u.cmd.Spreadsheet)
	if err != nil {
		return err
	}
	if file != nil {
		u.file = file
		return nil
	}

	dots.SetDescription(fmt.Sprintf("creating '%s'...", u.cmd.Spreadsheet))
	u.file, err = u.client.CreateSpreadsheet(u.ctx, u.cmd.Spreadsheet)
	return err
}

// REVIEW: why dos this exist?
// loadSpreadsheet fetches sheet metadata for the selected file.
func (u *upRunner) loadSpreadsheet() error {
	spreadsheet, err := u.client.GetSpreadsheet(u.ctx, u.file.ID)
	if err != nil {
		return err
	}
	u.spreadsheet = spreadsheet
	return nil
}

// readCSV reads and rectangularizes the input CSV.
func (c *UpCmd) readCSV() (google.Rows, error) {
	file, err := os.Open(c.CSVPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not found: %s", c.CSVPath)
		}
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("csv is empty: %s", c.CSVPath)
	}
	return google.Rectangularize(rows), nil
}
