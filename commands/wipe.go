package commands

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

	cmd.dots.SayWipeSpreadsheet(cmd.file.Name)
	if err := cmd.client.WipeSpreadsheet(cmd.ctx, cmd.file.ID); err != nil {
		return err
	}
	cmd.dots.SayWipedSpreadsheet(cmd.file.Name)
	return nil
}
