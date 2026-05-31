package commands

import (
	"fmt"
	"os"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
)

type DownCmd struct {
	Output      string `short:"o" type:"path" help:"Where to write the CSV."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
	Sheet       string `arg:"" optional:"" name:"sheet" help:"Sheet name."`
}

func (c *DownCmd) Run(_ *App) error {
	// fetch
	rows, err := c.run0()
	if err != nil {
		return err
	}

	// print
	writer := os.Stdout
	if c.Output != "" && c.Output != "-" {
		file, err := os.Create(c.Output)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}
	return util.CSVWrite(writer, rows)
}

func (c *DownCmd) run0() (rows google.Rows, err error) {
	//
	// init
	//

	cmd, err := srunStart(srunOptions{spreadsheet: c.Spreadsheet})
	if err != nil {
		return nil, err
	}
	defer func() { cmd.stop(err) }()

	//
	// find sheet
	//

	cmd.dots.SayFindSheet(c.Sheet)
	sheet, err := cmd.client.FindSheet(cmd.ctx, cmd.file.ID, c.Sheet)
	if err != nil {
		return nil, err
	}
	if sheet == nil {
		return nil, fmt.Errorf("found spreadsheet file '%s', but could not find sheet named '%s'", c.Spreadsheet, c.Sheet)
	}

	//
	// download
	//

	cmd.dots.SayDownloadRows(cmd.file.Name)
	rows, err = cmd.client.GetRows(cmd.ctx, cmd.file.ID, sheet.Title)
	if err != nil {
		return nil, err
	}
	isStdout := c.Output == "" || c.Output == "-"
	if !isStdout {
		cmd.dots.SaySaveRows(len(rows), c.Output)
	}
	return rows, nil
}
