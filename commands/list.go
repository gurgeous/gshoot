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
	client, err := google.NewClient(ctx, google.ReadOnlyScopes())
	if err != nil {
		return err
	}

	dots.SetDescription("getting list of spreadsheets...")
	files, err := client.ListSpreadsheets(ctx, 10)
	if err != nil {
		return err
	}
	dots.SetDescription(fmt.Sprintf("%d recent spreadsheets", len(files)))
	dots.Stop()

	// print
	for i, file := range files {
		const width = 30
		num := ux.Dim.Render(fmt.Sprintf("%2d.", i+1))
		name := fmt.Sprintf("%-"+strconv.Itoa(width)+"s", util.Truncate(file.Name, width))
		date := ux.Dim.Render(util.DateAndTimeStr(file.ModifiedByMeTime))
		fmt.Printf(
			" %s %s   %s\n",
			num,
			util.Hyperlink(os.Stdout, util.SpreadsheetURL(file.Id), name),
			date,
		)
	}

	return nil
}
