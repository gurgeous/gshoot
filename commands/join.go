package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/gurgeous/gshoot/gog"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

//
// Join a CSV into an existing sheet without replacing existing values.
//

type JoinCmd struct {
	Key         string   `required:"" help:"Column used to match rows."`
	Sheet       string   `help:"Destination sheet name."`
	Columns     []string `help:"Comma-separated CSV columns to join."`
	Force       bool     `short:"f" help:"Skip confirmation."`
	Spreadsheet string   `arg:"" name:"spreadsheet" help:"Spreadsheet name, ID, or URL."`
	CSVPath     string   `arg:"" name:"csv" type:"path" help:"CSV file to join."`
}

func (c *JoinCmd) Run() (err error) {
	right, err := util.CSVRead(c.CSVPath)
	if err != nil {
		return err
	}

	// fire up progress
	cmd, err := srunStart(os.Stderr, srunOptions{spreadsheet: c.Spreadsheet})
	if err != nil {
		return err
	}
	defer func() { cmd.stop(err) }()

	// find spreadsheet
	cmd.progress.SayFetchSpreadsheet(cmd.file.Name)
	spreadsheet, err := cmd.client.GetSpreadsheetWithGridData(cmd.ctx, cmd.file.ID)
	if err != nil {
		return err
	}
	var sheet *gog.Sheet
	if c.Sheet == "" {
		sheet = spreadsheet.Sheets[0]
	} else {
		sheet = findSheet(spreadsheet, c.Sheet)
	}
	if sheet == nil {
		return fmt.Errorf("sheet %q not found", c.Sheet)
	}

	// pull data
	left, err := cmd.client.GetRows(cmd.ctx, cmd.file.ID, sheet.Title)
	if err != nil {
		return err
	}

	// join
	join, err := newJoiner(left, gog.Rows(right), c.Key, c.Columns)
	if err != nil {
		return err
	}

	// preview/prompt
	cmd.stop(nil)
	join.preview(os.Stdout)
	if !c.Force {
		fmt.Fprintln(os.Stderr)
		prompt := ux.Warn.Render("join into '"+cmd.file.Name+" / "+sheet.Title+"'?") + " " + ux.Muted.Render("(y/n)")
		util.Confirm(prompt)
		fmt.Fprintln(os.Stderr)
	}

	// backup
	fmt.Fprintln(os.Stderr, "creating backup...")
	backupTitle := fmt.Sprintf("%s backup %s", cmd.file.Name, time.Now().UTC().Format("2006-01-02 150405 UTC"))
	backup, err := cmd.client.CopySpreadsheet(cmd.ctx, cmd.file.ID, backupTitle)
	if err != nil {
		return fmt.Errorf("create backup: %w", err)
	}
	backupURL := util.SpreadsheetURL(backup.ID) + "/edit"

	// apply
	cmd.progress = ux.StartProgress(os.Stderr, "joining...")
	hasFilter := spreadsheet.Data[sheet.ID].FilterRange != nil
	if _, err := cmd.client.Apply(cmd.ctx, cmd.file.ID, join.operations(sheet.ID, hasFilter)); err != nil {
		return fmt.Errorf("join failed: %w\nbackup: %s", err, backupURL)
	}
	cmd.stop(nil)

	// done!
	fmt.Printf("spreadsheet: %s/edit\nbackup: %s\n", util.SpreadsheetURL(cmd.file.ID), backupURL)
	return nil
}
