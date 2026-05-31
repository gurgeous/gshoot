package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

type ListCmd struct{}

func (c *ListCmd) Run(a *App) error {
	// fetch
	files, err := c.run0()
	if err != nil {
		return err
	}

	// print
	for i, file := range files {
		num := ux.Muted.Render(fmt.Sprintf("%2d.", i+1))
		name := fmt.Sprintf("%-30s", util.Truncate(file.Name, 30))
		date := ux.Muted.Render(util.DateAndTimeStr(file.ModifiedByMeTime))
		link := a.Hyperlink(util.SpreadsheetURL(file.ID), name)
		a.Printf(" %s %s     %s\n", num, link, date)
	}

	return nil
}

func (c *ListCmd) run0() (files []*google.File, err error) {
	dots := ux.StartDots(os.Stderr, "connecting to Google Sheets...")
	defer func() {
		if err == nil {
			dots.Stop()
		} else {
			dots.Cancel()
		}
	}()

	ctx := context.Background()
	client, err := google.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	dots.SayListFiles()
	files, err = client.ListSpreadsheetFiles(ctx, 20)
	if err != nil {
		return nil, err
	}

	dots.SayListedSpreadsheets(len(files))
	return files, nil
}
