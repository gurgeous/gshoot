package sub

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
)

//
// pkg init
//

var outputPath string

func init() {
	downCommand := &cobra.Command{
		Use:   "down <spreadsheet> [sheet]",
		Short: "Download a Google Sheet as CSV",
		Example: strings.Join([]string{
			"gshoot down Budget",
			"  gshoot down Budget Q1 --output q1.csv",
		}, "\n"),
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) >= 1 && len(args) <= 2 {
				return nil
			}
			return fmt.Errorf("expected `gshoot down <spreadsheet> [sheet]`")
		},
		RunE: downHandler,
	}
	downCommand.Flags().StringVarP(&outputPath, "output", "o", "", "where to write the CSV")
	rootCmd.AddCommand(downCommand)
}

//
// handler
//

func downHandler(cmd *cobra.Command, args []string) error {
	// parse args
	spreadsheetName := args[0]
	sheetName := ""
	if len(args) == 2 {
		sheetName = args[1]
	}

	dots := ux.StartDots(cmd.ErrOrStderr(), "opening Google Sheets...")
	var rows google.Rows
	{
		defer dots.Stop()

		// auth
		ctx := context.Background()
		client, err := google.NewClient(ctx, google.ReadOnlyScopes())
		if err != nil {
			return err
		}

		// fetch
		dots.SetDescription("finding spreadsheet...")
		spreadsheet, err := client.FindSpreadsheet(ctx, spreadsheetName)
		if err != nil {
			return fmt.Errorf("could not find spreadsheet '%s'", spreadsheetName)
		}

		// now find sheet
		dots.SetDescription("finding that sheet...")
		sheet, err := client.FindSheet(ctx, spreadsheet.Id, sheetName)
		if err != nil {
			return err
		}
		if sheet == nil {
			return fmt.Errorf("in spreadsheet '%s', could not find sheet '%s'", spreadsheetName, sheetName)
		}

		// download
		dots.SetDescription("downloading rows...")
		rows, err = client.GetRows(ctx, spreadsheet.Id, sheet)
		if err != nil {
			return err
		}
		if outputPath != "" {
			dots.SetDescription(fmt.Sprintf("saving %s", outputPath))
		}
	}

	// write
	var writer io.Writer
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	} else {
		writer = cmd.OutOrStdout()
	}

	return util.CSVWrite(writer, rows)
}
