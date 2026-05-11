package sub

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gurgeous/gshoot/internal/down"
	"github.com/gurgeous/gshoot/internal/google"
	"github.com/spf13/cobra"
)

var (
	downloadSheet = down.Download

	outputPath string
)

func init() {
	cmd := &cobra.Command{
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
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "where to write the CSV")
	rootCmd.AddCommand(cmd)
}

func DownHandler(cmd *cobra.Command, args []string) error {
	sheetName := ""
	if len(args) == 2 {
		sheetName = args[1]
	}

	ctx := context.Background()
	client, err := google.NewClient(ctx, google.ReadOnlyScopes())
	if err != nil {
		return err
	}
	values, err := downloadSheet(ctx, client, args[0], sheetName)
	if err != nil {
		return err
	}

	writer := cmd.OutOrStdout()
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	return down.WriteCSV(writer, values)
}
