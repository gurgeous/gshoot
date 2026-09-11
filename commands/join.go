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
	spreadsheet, err := cmd.client.GetSpreadsheet(cmd.ctx, cmd.file.ID)
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

	// backup tab
	fmt.Fprintln(os.Stderr, "creating backup tab...")
	backupTitle := fmt.Sprintf("%s backup %s", sheet.Title, time.Now().UTC().Format("2006-01-02 150405 UTC"))
	backup, err := cmd.client.DuplicateTab(cmd.ctx, cmd.file.ID, sheet.Title, backupTitle)
	if err != nil {
		return fmt.Errorf("create backup tab: %w", err)
	}
	backupURL := fmt.Sprintf("%s/edit#gid=%d", util.SpreadsheetURL(cmd.file.ID), backup.ID)

	// apply in ordered batches
	hasFilter := spreadsheet.Data[sheet.ID].FilterRange != nil
	plan := join.operations(sheet.ID, hasFilter)
	columns := len(join.insertedColumns())
	prepare := fmt.Sprintf("preparing %d columns", columns)
	if join.rowCounts.right > 0 {
		rowLabel := "rows"
		if join.rowCounts.right == 1 {
			rowLabel = "row"
		}
		prepare += fmt.Sprintf(" and %d %s", join.rowCounts.right, rowLabel)
	}
	finish := fmt.Sprintf("resizing %d columns", columns)
	if hasFilter {
		finish = "updating filter and " + finish
	}
	phases := []struct {
		description string
		operations  []gog.Operation
	}{
		{description: prepare, operations: plan.prepare},
		{description: fmt.Sprintf("writing %d rows across %d ranges", len(join.rows), len(plan.values)), operations: plan.values},
		{description: finish, operations: plan.finish},
	}
	for i, phase := range phases {
		label := fmt.Sprintf("joining (%d/3): %s...", i+1, phase.description)
		cmd.progress = ux.StartProgress(os.Stderr, label)
		if err := cmd.client.ApplySheetBatch(cmd.ctx, cmd.file.ID, sheet, phase.operations); err != nil {
			return fmt.Errorf("join phase %d/3 (%s) failed: %w\nbackup: %s", i+1, phase.description, err, backupURL)
		}
		cmd.stop(nil)
	}

	// done!
	fmt.Printf("spreadsheet: %s/edit\nbackup: %s\n", util.SpreadsheetURL(cmd.file.ID), backupURL)
	return nil
}
