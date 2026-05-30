package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/ux"
)

// WipeCmd resets a spreadsheet to one blank Sheet1.
type WipeCmd struct {
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
}

// Run wipes the selected spreadsheet, creating it if needed.
func (c *WipeCmd) Run() error {
	ctx := context.Background()
	dots := ux.StartDots(os.Stderr, "connecting to Google Sheets...")
	defer dots.Stop()

	client, err := google.NewClient(ctx)
	if err != nil {
		return err
	}

	dots.SetDescription(fmt.Sprintf("find or create spreadsheet '%s'...", c.Spreadsheet))
	file, err := client.FindOrCreateSpreadsheetFile(ctx, c.Spreadsheet)
	if err != nil {
		return err
	}
	dots.SetDescription("wiping spreadsheet...")
	if err := client.WipeSpreadsheet(ctx, file.ID); err != nil {
		return err
	}
	dots.SetDescription("wiped " + file.Name)

	fmt.Println("wiped " + file.Name)
	return nil
}
