package commands

import (
	"fmt"
)

// WipeCmd resets a spreadsheet to one blank Sheet1.
type WipeCmd struct {
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
}

// Run wipes the selected spreadsheet, creating it if needed.
func (c *WipeCmd) Run() error {
	cmd, err := srunStart(srunOptions{spreadsheet: c.Spreadsheet, create: true})
	if err != nil {
		return err
	}
	defer cmd.stop()

	cmd.dots.SetDescription("wiping spreadsheet...")
	if err := cmd.client.WipeSpreadsheet(cmd.ctx, cmd.file.ID); err != nil {
		return err
	}
	cmd.dots.SetDescription("wiped " + cmd.file.Name)

	fmt.Println("wiped " + cmd.file.Name)
	return nil
}
