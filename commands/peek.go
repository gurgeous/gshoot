package commands

import (
	"fmt"

	"github.com/gurgeous/gshoot/app"
	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

// PeekCmd lists sheet names in a spreadsheet.
type PeekCmd struct {
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
}

// Run prints one sheet name per line.
func (c *PeekCmd) Run(a *app.App) error {
	sheets, err := c.run0(a)
	if err != nil {
		return err
	}
	for ii, sheet := range sheets {
		num := ux.Muted.Render(fmt.Sprintf("%2d.", ii+1))
		rows := ux.Success.Render(util.FormatInt(sheet.GridProperties.RowCount))
		cols := ux.Success.Render(util.FormatInt(sheet.GridProperties.ColumnCount))
		x := ux.Muted.Render("x")
		a.Printf("%s %-25s %s %s %s\n", num, sheet.Title, rows, x, cols)
	}
	return nil
}

func (c *PeekCmd) run0(a *app.App) (sheets []*google.Sheet, err error) {
	cmd, err := srunStart(a, srunOptions{spreadsheet: c.Spreadsheet})
	if err != nil {
		return nil, err
	}
	defer func() { cmd.stop(err) }()

	cmd.dots.SayPeekSheets(cmd.file.Name)
	sheets, err = cmd.client.GetSheets(cmd.ctx, cmd.file.ID)
	if err != nil {
		return nil, err
	}

	return sheets, nil
}
