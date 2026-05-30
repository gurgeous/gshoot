package commands

import (
	"context"
	"fmt"

	"github.com/gurgeous/gshoot/google"
)

// WipeCmd resets a spreadsheet to one blank Sheet1.
type WipeCmd struct {
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
}

// Run wipes the selected spreadsheet, creating it if needed.
func (c *WipeCmd) Run() error {
	ctx := context.Background()
	client, err := google.NewClient(ctx)
	if err != nil {
		return err
	}

	file, err := client.FindOrCreateSpreadsheetFile(ctx, c.Spreadsheet)
	if err != nil {
		return err
	}
	if err := client.WipeSpreadsheet(ctx, file.ID); err != nil {
		return err
	}
	fmt.Println("wiped " + file.Name)
	return nil
}
