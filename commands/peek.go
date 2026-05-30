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

// PeekCmd lists sheet names in a spreadsheet.
type PeekCmd struct {
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
}

// Run prints one sheet name per line.
func (c *PeekCmd) Run() error {
	ctx := context.Background()
	dots := ux.StartDots(os.Stderr, "connecting to Google Sheets...")
	defer stopDots(&dots)

	client, err := google.NewClient(ctx)
	if err != nil {
		return err
	}

	dots.SetDescription("finding spreadsheet file...")
	file, err := client.FindSpreadsheetFile(ctx, c.Spreadsheet)
	if err != nil {
		return err
	}
	if file == nil {
		return errors.New("could not find spreadsheet file '" + c.Spreadsheet + "'")
	}

	dots.SetDescription("getting sheet list...")
	sheets, err := client.GetSheets(ctx, file.ID)
	if err != nil {
		return err
	}
	dots.SetDescription(fmt.Sprintf("%d sheets in %s", len(sheets), file.Name))
	stopDots(&dots)

	for ii, sheet := range sheets {
		rows := ux.Success.Render(util.FormatInt(sheet.GridProperties.RowCount))
		cols := ux.Success.Render(util.FormatInt(sheet.GridProperties.ColumnCount))
		mul := ux.Muted.Render("x")
		fmt.Printf("%2d. %-20s %s %s %s\n", ii+1, sheet.Title, rows, mul, cols)
	}
	return nil
}

func stopDots(dots **ux.Dots) {
	if *dots == nil {
		return
	}
	(*dots).Stop()
	*dots = nil
}
