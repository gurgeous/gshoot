package commands

import (
	"fmt"
	"os"
	"slices"

	"github.com/gurgeous/gshoot/gog"
	"github.com/gurgeous/gshoot/util"
)

//
// Append CSV data rows to a sheet with identical columns.
//

type AppendCmd struct {
	Sheet       string `help:"Destination sheet name."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name, ID, or URL."`
	CSVPath     string `arg:"" name:"csv" type:"path" help:"CSV file to append."`
}

func (c *AppendCmd) Run() (err error) {
	rows, err := util.CSVRead(c.CSVPath)
	if err != nil {
		return err
	}
	if err := gog.Rows(rows).ValidateHeaders("csv"); err != nil {
		return err
	}

	cmd, err := srunStart(os.Stderr, srunOptions{spreadsheet: c.Spreadsheet})
	if err != nil {
		return err
	}
	defer func() { cmd.stop(err) }()

	cmd.progress.SayFindSheet(c.Sheet)
	sheet, err := cmd.client.FindSheet(cmd.ctx, cmd.file.ID, c.Sheet)
	if err != nil {
		return err
	}
	if sheet == nil {
		return fmt.Errorf("sheet %q not found", c.Sheet)
	}

	header, err := cmd.client.GetHeader(cmd.ctx, cmd.file.ID, sheet.Title)
	if err != nil {
		return err
	}
	if !slices.Equal(rows[0], header) {
		return fmt.Errorf("csv columns must exactly match sheet %q", sheet.Title)
	}

	cmd.progress.SayAppendRows(len(rows)-1, cmd.file.Name, sheet.Title)
	if len(rows) > 1 {
		if err := cmd.client.AppendRows(cmd.ctx, cmd.file.ID, sheet.Title, rows[1:]); err != nil {
			return err
		}
	}

	fmt.Printf("%s/edit#gid=%d\n", util.SpreadsheetURL(cmd.file.ID), sheet.ID)
	return nil
}
