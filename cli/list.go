package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"google.golang.org/api/drive/v3"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

type ListCmd struct{}

func (c *ListCmd) Run(app *app) error {
	ctx := context.Background()
	dots := ux.StartDots(app.stderr, "connecting to Google Sheets...")
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

	printFiles(app.stdout, files)
	return nil
}

func printFiles(w io.Writer, files []*drive.File) {
	for i, file := range files {
		const width = 30
		num := ux.Dim.Render(fmt.Sprintf("%2d.", i+1))
		name := fmt.Sprintf("%-"+strconv.Itoa(width)+"s", util.Truncate(file.Name, width))
		date := ux.Dim.Render(util.DateAndTimeStr(file.ModifiedByMeTime))
		fmt.Fprintf(
			w,
			" %s %s   %s\n",
			num,
			util.Hyperlink(w, util.SpreadsheetURL(file.Id), name),
			date,
		)
	}
}
