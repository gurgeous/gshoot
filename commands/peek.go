package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/gurgeous/gshoot/google"
)

// PeekCmd lists sheet names in a spreadsheet.
type PeekCmd struct {
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
}

// Run prints one sheet name per line.
func (c *PeekCmd) Run() error {
	ctx := context.Background()
	client, err := google.NewClient(ctx)
	if err != nil {
		return err
	}

	file, err := client.FindSpreadsheetFile(ctx, c.Spreadsheet)
	if err != nil {
		return err
	}
	if file == nil {
		return errors.New("could not find spreadsheet file '" + c.Spreadsheet + "'")
	}

	sheets, err := client.GetSheets(ctx, file.ID)
	if err != nil {
		return err
	}
	for _, sheet := range sheets {
		fmt.Printf("%s %dx%d\n", sheet.Title, sheet.GridProperties.RowCount, sheet.GridProperties.ColumnCount)
	}
	return nil
}
