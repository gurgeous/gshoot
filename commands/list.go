package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

type ListCmd struct{}

func (c *ListCmd) Run() error {
	ctx := context.Background()
	dots := ux.StartDots(os.Stderr, "connecting to Google Sheets...")
	client, err := google.NewClient(ctx)
	if err != nil {
		return err
	}

	dots.SetDescription("getting list of spreadsheets...")
	files, err := client.ListSpreadsheets(ctx, 20)
	if err != nil {
		return err
	}
	dots.SetDescription(fmt.Sprintf("%d most recent spreadsheets", len(files)))
	dots.Stop()

	// print
	for i, file := range files {
		const width = 30
		num := ux.Muted.Render(fmt.Sprintf("%2d.", i+1))
		name := fmt.Sprintf("%-"+strconv.Itoa(width)+"s", util.Truncate(file.Name, width))
		date := ux.Muted.Render(util.DateAndTimeStr(file.ModifiedByMeTime))
		fmt.Printf(
			" %s %s     %s\n",
			num,
			util.Hyperlink(os.Stdout, util.SpreadsheetURL(file.ID), name),
			date,
		)
	}

	return nil
}
