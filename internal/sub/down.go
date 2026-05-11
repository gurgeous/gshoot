package sub

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gurgeous/gshoot/internal/down"
	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
)

var (
	downloadSheet = down.Download
	outputPath    string
	downCommand   = &cobra.Command{
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
		RunE: DownHandler,
	}
)

//
// pkg init
//

func init() {
	downCommand.Flags().StringVarP(&outputPath, "output", "o", "", "where to write the CSV")
	rootCmd.AddCommand(downCommand)
}

//
// handler
//

func DownHandler(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	// parse args
	sheetName := ""
	if len(args) == 2 {
		sheetName = args[1]
	}

	// auth
	dots := ux.StartDots(stderr, "opening Google Sheets...")
	client, err := google.NewClient(ctx, google.ReadOnlyScopes())
	if err != nil {
		return err
	}

	// fetch
	dots.SetDescription("fetching")
	rows, err := downloadSheet(ctx, client, args[0], sheetName)
	if err != nil {
		return err
	}
	if outputPath != "" {
		dots.SetDescription(fmt.Sprintf("saving %s", outputPath))
	}
	dots.Stop()

	// write
	writer := stdout
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}

	return util.CSVWrite(writer, rows)
}

//
// helpers
//
