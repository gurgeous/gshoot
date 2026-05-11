package sub

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
)

//
// pkg init
//

func init() {
	listCommand := &cobra.Command{
		Use:           "list",
		Short:         "List your Google Sheets",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          noArgs("gshoot list"),
		RunE:          ListHandler,
	}
	rootCmd.AddCommand(listCommand)
}

//
// command handler
//

func ListHandler(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	// auth
	dots := ux.StartDots(stderr, "opening Google Sheets...")
	client, err := google.NewClient(ctx, google.ReadOnlyScopes())
	if err != nil {
		return err
	}

	// fetch
	dots.SetDescription("fetching spreadsheets")

	res, err := client.Drive.Files.List().
		Context(ctx).
		Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
		OrderBy("modifiedTime desc,name").
		PageSize(int64(10)).
		Fields("files(id,name,modifiedTime)").
		Do()
	if err != nil {
		return err
	}
	dots.SetDescription(fmt.Sprintf("%d recent spreadsheets", len(res.Files)))
	dots.Stop()

	// print
	for i, file := range res.Files {
		const width = 30
		num := ux.Dim.Render(fmt.Sprintf("%2d.", i+1))
		name := fmt.Sprintf("%-"+strconv.Itoa(width)+"s", util.Truncate(file.Name, width))
		date := ux.Dim.Render(util.DateAndTimeStr(file.ModifiedTime))
		fmt.Fprintf(
			stdout,
			" %s %s   %s\n",
			num,
			util.Hyperlink(stdout, util.SpreadsheetURL(file.Id), name),
			date,
		)
	}

	return nil
}
