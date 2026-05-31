package commands

import (
	"fmt"

	"github.com/gurgeous/gshoot/app"
)

// WipeCmd resets a spreadsheet to one blank Sheet1.
type WipeCmd struct {
	Force       bool   `short:"f" help:"Skip confirmation."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
}

// Run wipes the selected spreadsheet, creating it if needed.
func (c *WipeCmd) Run(a *app.App) error {
	if !c.Force {
		a.Confirm(fmt.Sprintf("wipe spreadsheet '%s'?", c.Spreadsheet))
	}

	cmd, err := srunStart(a, srunOptions{spreadsheet: c.Spreadsheet, create: true})
	if err != nil {
		return err
	}
	defer cmd.stop()

	cmd.dots.SayWipeSpreadsheet(cmd.file.Name)
	if err := cmd.client.WipeSpreadsheet(cmd.ctx, cmd.file.ID); err != nil {
		return err
	}
	cmd.dots.SayWipedSpreadsheet(cmd.file.Name)
	return nil
}
