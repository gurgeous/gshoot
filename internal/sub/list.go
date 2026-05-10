package sub

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
	"google.golang.org/api/drive/v3"
)

func init() { rootCmd.AddCommand(newListCommand()) }

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List your Google Sheets",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          noArgs("gshoot list"),
		RunE:          run,
	}
	return cmd
}

//
// guts of command
//

func run(cmd *cobra.Command, _ []string) error {
	dots := ux.StartDots(cmd.ErrOrStderr(), "gshoot: opening Google Sheets...")
	ctx := context.Background()

	// auth
	client, err := google.NewClient(ctx, google.ReadOnlyScopes())
	if err != nil {
		return err
	}

	// list
	files, err := recent(ctx, client, 10)
	if err != nil {
		dots.SetDescription("list failed")
		dots.Stop()
		return err
	}

	// done, print
	dots.SetDescription(fmt.Sprintf("%d recent spreadsheets", len(files)))
	dots.Stop()
	printFiles(cmd.OutOrStdout(), files)

	return nil
}

// Recent returns recent spreadsheets ordered by modified time.
func recent(ctx context.Context, client *google.Client, limit int) ([]*drive.File, error) {
	res, err := client.Drive.Files.List().
		Context(ctx).
		Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
		OrderBy("modifiedTime desc,name").
		PageSize(int64(limit)).
		Fields("files(id,name,modifiedTime)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list spreadsheets: %w", err)
	}
	return res.Files, nil
}

// now print the results
func printFiles(out io.Writer, files []*drive.File) {
	for i, file := range files {
		const width = 30
		num := ux.Dim.Render(fmt.Sprintf("%2d.", i+1))
		name := fmt.Sprintf("%-"+strconv.Itoa(width)+"s", util.Truncate(file.Name, width))
		date := ux.Dim.Render(util.DateAndTimeStr(file.ModifiedTime))
		fmt.Fprintf(
			out,
			" %s %s   %s\n",
			num,
			util.Hyperlink(out, util.SpreadsheetURL(file.Id), name),
			date,
		)
	}
}
