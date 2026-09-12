package commands

import (
	"fmt"
	"os"

	"github.com/gurgeous/gshoot/util"
)

//
// Transform plaintext cells into links backed up in an adjacent column.
//

type HyperlinkCmd struct {
	Sheet       string `help:"Destination sheet name."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name, ID, or URL."`
	TextColumn  string `arg:"" name:"plaintext-column" help:"Column containing hyperlink text."`
	LinkColumn  string `arg:"" optional:"" name:"link-column" help:"Column containing hyperlink targets."`
}

func (c *HyperlinkCmd) Run() (err error) {
	cmd, err := srunStart(os.Stderr, srunOptions{spreadsheet: c.Spreadsheet})
	if err != nil {
		return err
	}
	defer func() { cmd.stop(err) }()

	// Read and validate.
	cmd.progress.SayFindSheet(c.Sheet)
	sheet, err := cmd.client.FindSheet(cmd.ctx, cmd.file.ID, c.Sheet)
	if err != nil {
		return err
	}
	if sheet == nil {
		return fmt.Errorf("sheet %q not found", c.Sheet)
	}
	rows, err := cmd.client.GetRows(cmd.ctx, cmd.file.ID, sheet.Title)
	if err != nil {
		return err
	}
	plan, err := newHyperlinker(rows, c.TextColumn, c.LinkColumn)
	if err != nil {
		return err
	}
	if len(plan.cells) == 0 {
		fmt.Printf("transformed: 0\n%s/edit#gid=%d\n", util.SpreadsheetURL(cmd.file.ID), sheet.ID)
		return nil
	}

	// Back up, then transform.
	cmd.progress.SayBackupColumn(c.TextColumn, plan.backupHeader)
	if err := cmd.client.ApplySheetBatch(cmd.ctx, cmd.file.ID, sheet, plan.prepare(sheet.ID)); err != nil {
		return fmt.Errorf("create backup column: %w", err)
	}
	cmd.progress.SayHyperlinkRows(len(plan.cells))
	if err := cmd.client.ApplySheetBatch(cmd.ctx, cmd.file.ID, sheet, plan.values(sheet.ID)); err != nil {
		return fmt.Errorf("write hyperlinks (backup column %q is intact): %w", plan.backupHeader, err)
	}

	fmt.Printf("transformed: %d\n%s/edit#gid=%d\n", len(plan.cells), util.SpreadsheetURL(cmd.file.ID), sheet.ID)
	return nil
}
