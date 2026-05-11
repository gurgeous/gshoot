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

var (
	outputPath      string
	spreadsheetName string
	sheetName       string
)

func init() {
	downCommand := &cobra.Command{
		Use:   "down <spreadsheet> [sheet]",
		Short: "Download a Google Sheet as CSV",
		Example: strings.Join([]string{
			"gshoot down Budget                      # output first sheet",
			"gshoot down Budget Q1 --output q1.csv   # save sheet q1 to q1.csv",
		}, "\n"),
		Args: downArgs,
		RunE: downHandler,
	}
	downCommand.Flags().StringVarP(&outputPath, "output", "o", "", "where to write the CSV")
	rootCmd.AddCommand(downCommand)
}

//
// handler
//

func downArgs(_ *cobra.Command, args []string) error {
	if len(args) == 0 || len(args) > 2 {
		return fmt.Errorf("expected `gshoot down <spreadsheet> [sheet]`")
	}
	spreadsheetName = args[0]
	if len(args) == 2 {
		sheetName = args[1]
	}
	return nil
}

func downHandler(cmd *cobra.Command, _ []string) error {
	var rows google.Rows
	{
		dots := ux.StartDots(cmd.ErrOrStderr(), "connecting to Google Sheets...")
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
		dots.SetDescription("finding specific sheet...")
		sheet, err := client.FindSheet(ctx, spreadsheet.Id, sheetName)
		if err != nil {
			return err
		}
		if sheet == nil {
			return fmt.Errorf("in spreadsheet '%s', could not find sheet '%s'", spreadsheetName, sheetName)
		}

		// download
		dots.SetDescription("downloading cells...")
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
