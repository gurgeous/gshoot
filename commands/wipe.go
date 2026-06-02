package commands

import (
	"os"

	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

//
// Wipe a spreadsheet back to one blank Sheet1
//

type WipeCmd struct {
	Force       bool   `short:"f" help:"Skip confirmation."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet file name."`
}

func (c *WipeCmd) Run() (err error) {
	if !c.Force {
		prompt := ux.Warn.Render("wipe spreadsheet '"+c.Spreadsheet+"'?") + " " + ux.Muted.Render("(y/n)")
		util.Confirm(prompt)
	}

	cmd, err := srunStart(os.Stderr, srunOptions{spreadsheet: c.Spreadsheet, create: true})
	if err != nil {
		return err
	}
	defer func() { cmd.stop(err) }()

	cmd.progress.SayWipeSpreadsheet(cmd.file.Name)
	if err = cmd.client.WipeSpreadsheet(cmd.ctx, cmd.file.ID); err != nil {
		return err
	}
	return nil
}
