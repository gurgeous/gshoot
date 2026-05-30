package commands

import (
	"fmt"

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
	sheets, err := c.run0()
	if err != nil {
		return err
	}
	for ii, sheet := range sheets {
		rows := ux.Success.Render(util.FormatInt(sheet.GridProperties.RowCount))
		cols := ux.Success.Render(util.FormatInt(sheet.GridProperties.ColumnCount))
		mul := ux.Muted.Render("x")
		fmt.Printf("%2d. %-20s %s %s %s\n", ii+1, sheet.Title, rows, mul, cols)
	}
	return nil
}

func (c *PeekCmd) run0() ([]*google.Sheet, error) {
	cmd, err := srunStart(srunOptions{spreadsheet: c.Spreadsheet})
	if err != nil {
		return nil, err
	}
	defer cmd.stop()

	cmd.dots.SetDescription("getting sheet list...")
	sheets, err := cmd.client.GetSheets(cmd.ctx, cmd.file.ID)
	if err != nil {
		return nil, err
	}
	cmd.dots.SetDescription(fmt.Sprintf("%d sheets in %s", len(sheets), cmd.file.Name))
	return sheets, nil
}
